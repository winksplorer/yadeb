# yadeb
Yet Another .deb Fetcher

## TODO

- [X] Basic CLI
    - [X] Help menu
- [ ] Install command
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
    - [ ] Fix things being marked as installed even when apt fails
    - [ ] Allow user to choose candidate if filtering doesn't work correctly
    - [ ] Allow user to disable filtering and directly choosing a candidate
    - [ ] Installing multiple packages at once
- [ ] Remove command
- [ ] Purge command
- [ ] Upgrade command
- [ ] Upgrade-all command
- [ ] List command
- [ ] Selfhost command
- [ ] Pin command