# Ubuntu 22.04 LTS (Jammy Jellyfish)
FROM photoprism/develop:230320-jammy

## Alternative Environments:
# FROM photoprism/develop:bookworm # Debian 12 (Bookworm)
# FROM photoprism/develop:bullseye # Debian 11 (Bullseye)
# FROM photoprism/develop:buster   # Debian 10 (Buster)
# FROM photoprism/develop:impish   # Ubuntu 21.10 (Impish Indri)
# FROM photoprism/develop:jammy    # Ubuntu 22.04 LTS (Jammy Jellyfish)

# Set default working directory.
WORKDIR "/go/src/github.com/photoprism/photoprism"
