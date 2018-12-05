
Example 

```golang
package main

import (
	ld "github.com/holgerarendt/lambda-dialogflow"
)

func hello(agent *ld.Agent) {
	name := agent.GetStringParam("name")
	agent.Say("Hello, " + name)
}

func main() {
	ld.Register("hello", hello)
	ld.Start()
}
```