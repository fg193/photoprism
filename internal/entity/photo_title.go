package entity

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"

	"github.com/photoprism/photoprism/internal/classify"
	"github.com/photoprism/photoprism/pkg/clean"
	"github.com/photoprism/photoprism/pkg/fs"
	"github.com/photoprism/photoprism/pkg/txt"
)

// HasTitle checks if the photo has a title.
func (m *Photo) HasTitle() bool {
	return m.PhotoTitle != ""
}

// NoTitle checks if the photo has no Title
func (m *Photo) NoTitle() bool {
	return m.PhotoTitle == ""
}

// SetTitle changes the photo title and clips it to 300 characters.
func (m *Photo) SetTitle(title, source string) {
	title = strings.Trim(title, "_&|{}<>: \n\r\t\\")
	title = strings.ReplaceAll(title, "\"", "'")
	title = txt.Shorten(title, txt.ClipLongName, txt.Ellipsis)

	if title == "" {
		return
	}

	if (SrcPriority[source] < SrcPriority[m.TitleSrc]) && m.HasTitle() {
		return
	}

	m.PhotoTitle = title
	m.TitleSrc = source
}

// UpdateTitle updated the photo title based on location and labels.
func (m *Photo) UpdateTitle(labels classify.Labels) error {
	if m.TitleSrc != SrcAuto && m.HasTitle() {
		return fmt.Errorf("photo: %s keeps existing %s title", m.String(), SrcString(m.TitleSrc))
	}

	var names string

	start := time.Now()
	oldTitle := m.PhotoTitle
	fileTitle := m.FileTitle()

	people := m.SubjectNames()
	m.UpdateDescription(people)
	if n := len(people); n > 0 && n < 4 {
		names = txt.JoinNames(people, true)
		log.Debugf("photo: %s title based on %s (%s)", m.String(), english.Plural(len(people), "person", "people"), clean.Log(names))
	}

	m.updateTitleByLocationAndNames(names)
	if m.NoTitle() {
		// TODO: User defined title format
		if names != "" {
			if len([]rune(names)) <= 35 && m.TakenSrc != SrcAuto {
				m.SetTitle(fmt.Sprintf("%s / %s", names, m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(names, SrcAuto)
			}
		} else if fileTitle == "" && len(labels) > 0 && labels[0].Priority >= -1 && labels[0].Uncertainty <= 85 && labels[0].Name != "" {
			if m.TakenSrc != SrcAuto {
				m.SetTitle(fmt.Sprintf("%s / %s", txt.Title(labels[0].Name), m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(txt.Title(labels[0].Name), SrcAuto)
			}
		} else if fileTitle != "" && len(fileTitle) <= 20 && !m.TakenAtLocal.IsZero() && m.TakenSrc != SrcAuto {
			m.SetTitle(fmt.Sprintf("%s / %s", fileTitle, m.TakenAtLocal.Format("2006")), SrcAuto)
		} else if fileTitle != "" {
			m.SetTitle(fileTitle, SrcAuto)
		} else {
			if m.TakenSrc != SrcAuto {
				m.SetTitle(fmt.Sprintf("%s / %s", m.PhotoName, m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(m.PhotoName, SrcAuto)
			}
		}
	}

	if m.PhotoTitle != oldTitle {
		log.Debugf("photo: %s has new title %s [%s]", m.String(), clean.Log(m.PhotoTitle), time.Since(start))
	}

	return nil
}

func (m *Photo) updateTitleByLocationAndNames(names string) {
	if !m.LocationLoaded() {
		return
	}
	loc := m.Cell

	if names == "" {
		m.updateTitleByLocation()
	} else if l := len([]rune(names)); l > 35 {
		m.SetTitle(names, SrcAuto)
	} else if l > 20 && (loc.NoCity() || loc.LongCity()) {
		m.SetTitle(fmt.Sprintf("%s / %s", names, m.TakenAt.Format("2006")), SrcAuto)
	} else if l > 20 {
		m.SetTitle(fmt.Sprintf("%s / %s", names, loc.City()), SrcAuto)
	} else if !loc.NoCity() && !loc.LongCity() {
		m.SetTitle(fmt.Sprintf("%s / %s / %s", names, loc.City(), m.TakenAt.Format("2006")), SrcAuto)
	} else if !loc.NoState() {
		m.SetTitle(fmt.Sprintf("%s / %s / %s", names, loc.State(), m.TakenAt.Format("2006")), SrcAuto)
	} else {
		m.SetTitle(fmt.Sprintf("%s / %s", names, m.TakenAt.Format("2006")), SrcAuto)
	}
	log.Debugf("photo: %s title %q based on location %+v", m.String(), m.PhotoTitle, loc)
}

func (m *Photo) updateTitleByLocation() {
	components := make([]string, 0, 5)
	for _, component := range []string{
		m.Cell.State(),
		m.Cell.City(),
		m.Cell.District(),
		m.Cell.Street(),
		m.Cell.Name(),
	} {
		if len(component) != 0 {
			components = append(components, component)
		}
	}
	if len(components) > 3 && len(components[len(components)-1]) > 24 {
		components = components[:len(components)-1]
	}
	components = m.uniqueTitleComponents(components)
	if len(components) > 3 {
		components = components[1:]
	}
	components = append(components, m.TakenAt.Format("2006"))
	m.SetTitle(strings.Join(components, " / "), SrcAuto)
}

func (_ *Photo) uniqueTitleComponents(arg []string) (ret []string) {
	ret = make([]string, 0, len(arg)+1)
	for i, short := range arg {
		include := true
		for j, long := range arg {
			if i != j && strings.Contains(long, short) {
				include = false
				break
			}
		}
		if include {
			ret = append(ret, short)
		}
	}
	return ret
}

// UpdateAndSaveTitle updates the photo title and saves it.
func (m *Photo) UpdateAndSaveTitle() error {
	if !m.HasID() {
		return fmt.Errorf("cannot save photo whithout id")
	}

	m.PhotoFaces = m.FaceCount()

	labels := m.ClassifyLabels()

	m.UpdateDateFields()

	if err := m.UpdateTitle(labels); err != nil {
		log.Info(err)
	}

	details := m.GetDetails()

	w := txt.UniqueWords(txt.Words(details.Keywords))
	w = append(w, labels.Keywords()...)
	details.Keywords = strings.Join(txt.UniqueWords(w), ", ")

	if err := m.IndexKeywords(); err != nil {
		log.Errorf("photo: %s", err.Error())
	}

	if err := m.Save(); err != nil {
		return err
	}

	return nil
}

// UpdateDescription updates the photo descriptions based on available metadata.
func (m *Photo) UpdateDescription(people []string) {
	if m.DescriptionSrc != SrcAuto {
		return
	}

	// Add subject names to description when there's more than one person.
	if len(people) > 3 {
		m.PhotoDescription = txt.JoinNames(people, false)
	} else {
		m.PhotoDescription = ""
	}
}

// FileTitle returns a photo title based on the file name and/or path.
func (m *Photo) FileTitle() string {
	// Generate title based on photo name, if not generated:
	if !fs.IsGenerated(m.PhotoName) {
		if title := txt.FileTitle(m.PhotoName); title != "" {
			return title
		}
	}

	// Generate title based on original file name, if any:
	if m.OriginalName != "" {
		if title := txt.FileTitle(m.OriginalName); !fs.IsGenerated(m.OriginalName) && title != "" {
			return title
		} else if title := txt.FileTitle(filepath.Dir(m.OriginalName)); title != "" {
			return title
		}
	}

	return ""
}

// SubjectNames returns all known subject names.
func (m *Photo) SubjectNames() []string {
	if f, err := m.PrimaryFile(); err == nil {
		return f.SubjectNames()
	}

	return nil
}

// SubjectKeywords returns keywords for all known subject names.
func (m *Photo) SubjectKeywords() []string {
	return txt.Words(strings.Join(m.SubjectNames(), " "))
}
