# dead :skull:
Dead simple live reloading for Go web servers

# TODO
                   
- [ ] Make it configurable what commands to run.
- [ ] Add documentation.
- [ ] Finish this TODO list.

```go
package main

import (
	"log"
	"net/http"

	"github.com/tomyl/dead"
)

func main() {
	dead.Default().Watch(".").Main()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```bash
$ go build
$ DEAD=watch ./demo
```
