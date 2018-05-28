# dead :skull:
Dead simple live reloading for Go web servers

# Usage

Just add a line of code like in example below:

```go
package main

import (
	"log"
	"net/http"

	"github.com/tomyl/dead"
)

func main() {
	dead.Default().Watch(".", "templates", "server/*").Main()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Now, if an enviroment variable `DEAD` (can be changed to something else) is set
to `watch` before starting the binary, `Main()` will watch the the specified
directories forever and re-invoke the binary with cleared ennviroment variable
(`Main()` will return immediately if the value isn't `watch`).

```bash
$ go build
$ DEAD=watch ./mybin
```

Whenever an `.html` file is modified in specified directories, the re-invoked
process is restarted. If a `.go` file is modified, `go build` (can be changed
to something else) is executed first.

# TODO
                   
- [ ] Make it configurable what extensions to trigger on.
- [ ] Add more documentation.
- [ ] Finish this TODO list.
