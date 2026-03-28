package engine

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/flosch/pongo2/v6"
	htmlTemplate "html/template"
)

func BenchmarkRenderStringSimple_Wisp(b *testing.B) {
	e := New()
	template := `Hello, {% .name%}!`
	data := map[string]interface{}{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithConditionals_Wisp(b *testing.B) {
	e := New()
	template := `{% if .show %}{% .content%}{% else %}hidden{% end %}`
	data := map[string]interface{}{
		"show":    true,
		"content": "Hello World",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithLoop_Wisp(b *testing.B) {
	e := New()
	template := `{% for .item in .items %}{% .item%}{% end %}`
	items := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		items[i] = "item"
	}
	data := map[string]interface{}{"items": items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithNestedAccess_Wisp(b *testing.B) {
	e := New()
	template := `{% .user.profile.name %}`
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithFilters_Wisp(b *testing.B) {
	e := New()
	template := `{% .name | upcase | truncate 10 %}`
	data := map[string]interface{}{"name": "Hello World Template"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringComplex_Wisp(b *testing.B) {
	e := New()
	template := `
<html>
<head><title>{% .title %}</title></head>
<body>
	<h1>{% .header %}</h1>
	{% if .show_list %}
	<ul>
		{% for .item in .items %}
		<li>{% .item.name%} - {% .item.price | currency %}</li>
		{% end %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% end %}
	<footer>{% .footer %}</footer>
</body>
</html>
`
	data := map[string]interface{}{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.00},
			map[string]interface{}{"name": "Item 2", "price": 20.00},
			map[string]interface{}{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkCaching_Wisp(b *testing.B) {
	e := New()
	template := `Hello, {% .name%}!`
	data := map[string]interface{}{"name": "World"}

	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkAutoEscape_Wisp(b *testing.B) {
	e := New()
	template := `{% .html %}`
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRegisterTemplate_Wisp(b *testing.B) {
	e := New()
	for i := 0; i < b.N; i++ {
		e.RegisterTemplate("test", `Hello, {% .name%}!`)
	}
}

func BenchmarkRenderFile_Wisp(b *testing.B) {
	e := New()
	e.RegisterTemplate("greeting", `Hello, {% .name%}!`)

	data := map[string]interface{}{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("greeting", data)
	}
}

func BenchmarkValidate_Wisp(b *testing.B) {
	e := New()
	template := `{% if .show %}{% .content%}{% elsif .alt %}alt{% else %}default{% end %}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Validate(template)
	}
}

func BenchmarkRenderStringSimpleWisp(b *testing.B) {
	e := New()
	tpl := `Hello, {{ .name }}!`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringSimpleTextTemplate(b *testing.B) {
	tpl := `Hello, {{ .name }}!`
	t, _ := template.New("test").Parse(tpl)
	data := map[string]interface{}{"name": "World"}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringSimpleHtmlTemplate(b *testing.B) {
	tpl := `Hello, {{ .name }}!`
	t, _ := htmlTemplate.New("test").Parse(tpl)
	data := map[string]interface{}{"name": "World"}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringSimplePongo2(b *testing.B) {
	tpl := `Hello, {{ name }}!`
	t, _ := pongo2.FromString(tpl)
	data := pongo2.Context{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkRenderStringWithConditionalsWisp(b *testing.B) {
	e := New()
	tpl := `{% if .show %}{% .content%}{% else %}hidden{% end %}`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{
		"show":    true,
		"content": "Hello World",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringWithConditionalsTextTemplate(b *testing.B) {
	tpl := `{{ if .show }}{{ .content }}{{ else }}hidden{{ end }}`
	t, _ := template.New("test").Parse(tpl)
	data := map[string]interface{}{
		"show":    true,
		"content": "Hello World",
	}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringWithConditionalsPongo2(b *testing.B) {
	tpl := `{% if show %}{{ content }}{% else %}hidden{% endif %}`
	t, _ := pongo2.FromString(tpl)
	data := pongo2.Context{
		"show":    true,
		"content": "Hello World",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkRenderStringWithLoopWisp(b *testing.B) {
	e := New()
	tpl := `{% for item in .items %}{% item %}{% end %}`
	e.RegisterTemplate("test", tpl)
	items := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		items[i] = "item"
	}
	data := map[string]interface{}{"items": items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringWithLoopTextTemplate(b *testing.B) {
	tpl := `{{ range .items }}{{ . }}{{ end }}`
	t, _ := template.New("test").Parse(tpl)
	items := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		items[i] = "item"
	}
	data := map[string]interface{}{"items": items}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringWithLoopPongo2(b *testing.B) {
	tpl := `{% for item in items %}{{ item }}{% endfor %}`
	t, _ := pongo2.FromString(tpl)
	items := make([]string, 100)
	for i := 0; i < 100; i++ {
		items[i] = "item"
	}
	data := pongo2.Context{"items": items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkRenderStringWithNestedAccessWisp(b *testing.B) {
	e := New()
	tpl := `{% .user.profile.name %}`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringWithNestedAccessTextTemplate(b *testing.B) {
	tpl := `{{ .user.profile.name }}`
	t, _ := template.New("test").Parse(tpl)
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John",
			},
		},
	}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringWithFiltersWisp(b *testing.B) {
	e := New()
	tpl := `{% .name | upcase | truncate 10 %}`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{"name": "Hello World Template"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringWithFiltersPongo2(b *testing.B) {
	tpl := `{{ name|upper|truncatechars:10 }}`
	t, _ := pongo2.FromString(tpl)
	data := pongo2.Context{"name": "Hello World Template"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkRenderStringComplexWisp(b *testing.B) {
	e := New()
	tpl := `
<html>
<head><title>{% .title %}</title></head>
<body>
	<h1>{% .header %}</h1>
	{% if .show_list %}
	<ul>
		{% for item in .items %}
		<li>{% item.name%} - {% item.price %}</li>
		{% endfor %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% end %}
	<footer>{% .footer %}</footer>
</body>
</html>
`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.00},
			map[string]interface{}{"name": "Item 2", "price": 20.00},
			map[string]interface{}{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkRenderStringComplexTextTemplate(b *testing.B) {
	tpl := `
<html>
<head><title>{{ .title }}</title></head>
<body>
	<h1>{{ .header }}</h1>
	{{ if .show_list }}
	<ul>
		{{ range .items }}
		<li>{{ .name }} - {{ .price }}</li>
		{{ end }}
	</ul>
	{{ else }}
	<p>No items available</p>
	{{ end }}
	<footer>{{ .footer }}</footer>
</body>
</html>
`
	t, _ := template.New("test").Parse(tpl)
	data := map[string]interface{}{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.00},
			map[string]interface{}{"name": "Item 2", "price": 20.00},
			map[string]interface{}{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkRenderStringComplexPongo2(b *testing.B) {
	tpl := `
<html>
<head><title>{{ title }}</title></head>
<body>
	<h1>{{ header }}</h1>
	{% if show_list %}
	<ul>
		{% for item in items %}
		<li>{{ item.name }} - {{ item.price }}</li>
		{% endfor %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% endif %}
	<footer>{{ footer }}</footer>
</body>
</html>
`
	t, _ := pongo2.FromString(tpl)
	data := pongo2.Context{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"items": []pongo2.Context{
			{"name": "Item 1", "price": 10.00},
			{"name": "Item 2", "price": 20.00},
			{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkAutoEscapeWisp(b *testing.B) {
	e := New()
	tpl := `{% .html %}`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkAutoEscapeHtmlTemplate(b *testing.B) {
	tpl := `{{ .html }}`
	t, _ := htmlTemplate.New("test").Parse(tpl)
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}
	buf := new(bytes.Buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkCachingWisp(b *testing.B) {
	e := New()
	tpl := `Hello, {{ .name }}!`
	e.RegisterTemplate("test", tpl)
	data := map[string]interface{}{"name": "World"}

	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("test", data)
	}
}

func BenchmarkCachingTextTemplate(b *testing.B) {
	tpl := `Hello, {{ .name }}!`
	t, _ := template.New("test").Parse(tpl)
	data := map[string]interface{}{"name": "World"}
	buf := new(bytes.Buffer)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkCachingPongo2(b *testing.B) {
	tpl := `Hello, {{ name }}!`
	t, _ := pongo2.FromString(tpl)
	data := pongo2.Context{"name": "World"}

	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}
