# jsondiff

`jsondiff` compares two JSON rawMessages and renders styled string of diff with rest of JSON object
The package builds a tree of diffs first, then renders it line by line. 

## Example

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/SayYoungMan/tfui/pkg/jsondiff"
)

func main() {
	before := json.RawMessage(`{"bucket":"uploads","tags":{"env":"dev"}}`)
	after := json.RawMessage(`{"bucket":"uploads","tags":{"env":"prod"}}`)

	out, err := jsondiff.Render(before, after, jsondiff.Options{})
	if err != nil {
		panic(err)
	}

	fmt.Print(out)
}
```

Output:

```diff
  {
    "bucket": "uploads",
    "tags": {
-     "env": "dev"
+     "env": "prod"
    }
  }
```

