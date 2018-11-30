
Example 

```golang
package main

import (
	ld "github.com/holgerarendt/lambda-dialogflow"
)

func hello(agent *ld.Agent) {
	agent.Say("Hello World")
}

func main() {
	ld.Register("hello", hello)
	ld.Start()
}
```