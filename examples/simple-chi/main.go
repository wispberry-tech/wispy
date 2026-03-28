package main

import (
	"fmt"
	"net/http"

	"template-wisp/pkg/engine"
)

func main() {
	e := engine.New()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		result, err := e.RenderString(`<html>
<body>
<h1>Hello, {% .name %}!</h1>
<p>Welcome to Wisp.</p>
</body>
</html>`, map[string]interface{}{
			"name": "World",
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, result)
	})

	fmt.Println("Server running at http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}
