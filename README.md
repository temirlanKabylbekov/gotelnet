# telnet utility
## Installation

```
go get -d github.com/temirlanKabylbekov/gotelnet
go install github.com/temirlanKabylbekov/gotelnet
```   

## Quick Start
- start to chat in both sides by typing in stdin
- press CTRL+D to finish chatting
```bash
nc -l 13370

make build
gotelnet -timeout=5s localhost 13370
```
