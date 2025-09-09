# yadeb

Yet Another .deb Fetcher

## TODO

- [X] Basic CLI
    - [X] Help menu
- [X] Install command
    - [X] Link parsing
    - [X] GitHub release fetching
    - [X] GitHub release parsing
    - [X] Basic candidate filtering
        - [X] *.deb
        - [X] CPU architecture
    - [X] Temp directory
    - [X] Downloading
    - [X] Marking as installed and keeping track
    - [X] Actually installing through apt
    - [X] Cleanup
    - [X] Downloading specific versions
    - [X] Fix things being marked as installed even when apt fails
    - [X] Allow user to choose candidate if filtering doesn't work correctly
    - [X] Use slices instead of maps (amateur hour code from me a week ago smh)
- [X] Remove/purge command
    - [X] Link parsing
    - [X] Actually removing through apt
    - [X] Removing installation mark
    - [X] Logging
    - [X] Purging
- [X] Upgrade command
    - [X] Link parsing
    - [X] Checking if latest is already installed
    - [X] Marking as updated
    - [X] Logging
    - [X] Cleanup
- [ ] Upgrade-all command
- [ ] List command