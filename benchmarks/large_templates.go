package benchmarks

import "fmt"

// Large template scenarios for wall-clock timing benchmarks.
// These are significantly bigger than the micro-benchmark templates
// to simulate realistic production workloads.

// --- Scenario metadata ---

type TimingScenario struct {
	Name      string
	Templates map[string]string
	Data      func() map[string]any
}

func AllTimingScenarios() []TimingScenario {
	return []TimingScenario{
		{"Large Page", LargePageTemplates, NewLargePageData},
		{"Large Loop (100 items)", LargeLoopTemplates, NewLargeLoopData},
		{"Nested Loops (10x10)", NestedLoopTemplates, NewNestedLoopData},
		{"Complex Page", ComplexPageTemplates, NewComplexPageData},
	}
}

// --- Large Page: full HTML page with ~20 variable interpolations ---

var LargePageTemplates = map[string]string{
	EngGrove: `<!DOCTYPE html>
<html lang="{% lang %}">
<head>
  <meta charset="utf-8">
  <title>{% site_name %} — {% page_title %}</title>
  <meta name="description" content="{% meta_description %}">
  <meta name="author" content="{% meta_author %}">
  <link rel="canonical" href="{% canonical_url %}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{% site_name %}</a>
      <span class="tagline">{% tagline %}</span>
    </nav>
    <div class="user">Welcome, {% user_name %} ({% user_role %})</div>
  </header>
  <main>
    <article>
      <h1>{% page_title %}</h1>
      <p class="lead">{% lead_text %}</p>
      <div class="content">{% body_text %}</div>
      <footer>
        <span class="category">{% category %}</span>
        <time>{% published_date %}</time>
        <span class="reading-time">{% reading_time %} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {% copyright_year %} {% site_name %}. All rights reserved.</p>
    <p>{% footer_text %}</p>
  </footer>
</body>
</html>`,

	EngHTMLTemplate: `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8">
  <title>{{.SiteName}} — {{.PageTitle}}</title>
  <meta name="description" content="{{.MetaDescription}}">
  <meta name="author" content="{{.MetaAuthor}}">
  <link rel="canonical" href="{{.CanonicalURL}}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{{.SiteName}}</a>
      <span class="tagline">{{.Tagline}}</span>
    </nav>
    <div class="user">Welcome, {{.UserName}} ({{.UserRole}})</div>
  </header>
  <main>
    <article>
      <h1>{{.PageTitle}}</h1>
      <p class="lead">{{.LeadText}}</p>
      <div class="content">{{.BodyText}}</div>
      <footer>
        <span class="category">{{.Category}}</span>
        <time>{{.PublishedDate}}</time>
        <span class="reading-time">{{.ReadingTime}} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {{.CopyrightYear}} {{.SiteName}}. All rights reserved.</p>
    <p>{{.FooterText}}</p>
  </footer>
</body>
</html>`,

	EngTextTemplate: `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8">
  <title>{{.SiteName}} — {{.PageTitle}}</title>
  <meta name="description" content="{{.MetaDescription}}">
  <meta name="author" content="{{.MetaAuthor}}">
  <link rel="canonical" href="{{.CanonicalURL}}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{{.SiteName}}</a>
      <span class="tagline">{{.Tagline}}</span>
    </nav>
    <div class="user">Welcome, {{.UserName}} ({{.UserRole}})</div>
  </header>
  <main>
    <article>
      <h1>{{.PageTitle}}</h1>
      <p class="lead">{{.LeadText}}</p>
      <div class="content">{{.BodyText}}</div>
      <footer>
        <span class="category">{{.Category}}</span>
        <time>{{.PublishedDate}}</time>
        <span class="reading-time">{{.ReadingTime}} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {{.CopyrightYear}} {{.SiteName}}. All rights reserved.</p>
    <p>{{.FooterText}}</p>
  </footer>
</body>
</html>`,

	EngPongo2: `<!DOCTYPE html>
<html lang="{{ lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ site_name }} — {{ page_title }}</title>
  <meta name="description" content="{{ meta_description }}">
  <meta name="author" content="{{ meta_author }}">
  <link rel="canonical" href="{{ canonical_url }}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{{ site_name }}</a>
      <span class="tagline">{{ tagline }}</span>
    </nav>
    <div class="user">Welcome, {{ user_name }} ({{ user_role }})</div>
  </header>
  <main>
    <article>
      <h1>{{ page_title }}</h1>
      <p class="lead">{{ lead_text }}</p>
      <div class="content">{{ body_text }}</div>
      <footer>
        <span class="category">{{ category }}</span>
        <time>{{ published_date }}</time>
        <span class="reading-time">{{ reading_time }} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {{ copyright_year }} {{ site_name }}. All rights reserved.</p>
    <p>{{ footer_text }}</p>
  </footer>
</body>
</html>`,

	EngJet: `<!DOCTYPE html>
<html lang="{{ .Lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ .SiteName }} — {{ .PageTitle }}</title>
  <meta name="description" content="{{ .MetaDescription }}">
  <meta name="author" content="{{ .MetaAuthor }}">
  <link rel="canonical" href="{{ .CanonicalURL }}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{{ .SiteName }}</a>
      <span class="tagline">{{ .Tagline }}</span>
    </nav>
    <div class="user">Welcome, {{ .UserName }} ({{ .UserRole }})</div>
  </header>
  <main>
    <article>
      <h1>{{ .PageTitle }}</h1>
      <p class="lead">{{ .LeadText }}</p>
      <div class="content">{{ .BodyText }}</div>
      <footer>
        <span class="category">{{ .Category }}</span>
        <time>{{ .PublishedDate }}</time>
        <span class="reading-time">{{ .ReadingTime }} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {{ .CopyrightYear }} {{ .SiteName }}. All rights reserved.</p>
    <p>{{ .FooterText }}</p>
  </footer>
</body>
</html>`,

	EngLiquid: `<!DOCTYPE html>
<html lang="{{ lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ site_name }} — {{ page_title }}</title>
  <meta name="description" content="{{ meta_description }}">
  <meta name="author" content="{{ meta_author }}">
  <link rel="canonical" href="{{ canonical_url }}">
</head>
<body>
  <header>
    <nav>
      <a href="/" class="logo">{{ site_name }}</a>
      <span class="tagline">{{ tagline }}</span>
    </nav>
    <div class="user">Welcome, {{ user_name }} ({{ user_role }})</div>
  </header>
  <main>
    <article>
      <h1>{{ page_title }}</h1>
      <p class="lead">{{ lead_text }}</p>
      <div class="content">{{ body_text }}</div>
      <footer>
        <span class="category">{{ category }}</span>
        <time>{{ published_date }}</time>
        <span class="reading-time">{{ reading_time }} min read</span>
      </footer>
    </article>
  </main>
  <footer>
    <p>&copy; {{ copyright_year }} {{ site_name }}. All rights reserved.</p>
    <p>{{ footer_text }}</p>
  </footer>
</body>
</html>`,
}

// --- Large Loop: 100 product items ---

var LargeLoopTemplates = map[string]string{
	EngGrove: `<div class="products">
<For each={products} as="product">
<div class="product"><If test={product.on_sale}> on-sale</If>">
  <h3>{% product.name %}</h3>
  <p class="price">${% product.price %}</p>
  <p class="desc">{% product.description %}</p>
  <span class="category">{% product.category %}</span>
  <If test={product.in_stock}><span class="badge">In Stock</span><Else /><span class="badge out">Out of Stock</span></If>
</div>
</For>
</div>`,

	EngHTMLTemplate: `<div class="products">
{{range .Products}}
<div class="product{{if .OnSale}} on-sale{{end}}">
  <h3>{{.Name}}</h3>
  <p class="price">${{.Price}}</p>
  <p class="desc">{{.Description}}</p>
  <span class="category">{{.Category}}</span>
  {{if .InStock}}<span class="badge">In Stock</span>{{else}}<span class="badge out">Out of Stock</span>{{end}}
</div>
{{end}}
</div>`,

	EngTextTemplate: `<div class="products">
{{range .Products}}
<div class="product{{if .OnSale}} on-sale{{end}}">
  <h3>{{.Name}}</h3>
  <p class="price">${{.Price}}</p>
  <p class="desc">{{.Description}}</p>
  <span class="category">{{.Category}}</span>
  {{if .InStock}}<span class="badge">In Stock</span>{{else}}<span class="badge out">Out of Stock</span>{{end}}
</div>
{{end}}
</div>`,

	EngPongo2: `<div class="products">
{% for product in products %}
<div class="product{% if product.on_sale %} on-sale{% endif %}">
  <h3>{{ product.name }}</h3>
  <p class="price">${{ product.price }}</p>
  <p class="desc">{{ product.description }}</p>
  <span class="category">{{ product.category }}</span>
  {% if product.in_stock %}<span class="badge">In Stock</span>{% else %}<span class="badge out">Out of Stock</span>{% endif %}
</div>
{% endfor %}
</div>`,

	EngJet: `<div class="products">
{{range _, product := .Products}}
<div class="product{{if product.OnSale}} on-sale{{end}}">
  <h3>{{product.Name}}</h3>
  <p class="price">${{product.Price}}</p>
  <p class="desc">{{product.Description}}</p>
  <span class="category">{{product.Category}}</span>
  {{if product.InStock}}<span class="badge">In Stock</span>{{else}}<span class="badge out">Out of Stock</span>{{end}}
</div>
{{end}}
</div>`,

	EngLiquid: `<div class="products">
{% for product in products %}
<div class="product{% if product.on_sale %} on-sale{% endif %}">
  <h3>{{ product.name }}</h3>
  <p class="price">${{ product.price }}</p>
  <p class="desc">{{ product.description }}</p>
  <span class="category">{{ product.category }}</span>
  {% if product.in_stock %}<span class="badge">In Stock</span>{% else %}<span class="badge out">Out of Stock</span>{% endif %}
</div>
{% endfor %}
</div>`,
}

// --- Nested Loops: categories with products ---

var NestedLoopTemplates = map[string]string{
	EngGrove: `<div class="catalog">
<For each={categories} as="cat">
<section class="category">
  <h2>{% cat.name %} ({% cat.count %} items)</h2>
  <div class="items">
  <For each={cat.products} as="product">
    <div class="product"><If test={product.featured}> featured</If>">
      <span class="name">{% product.name %}</span>
      <span class="price">${% product.price %}</span>
      <If test={product.on_sale}><span class="sale">SALE</span></If>
    </div>
  </For>
  </div>
</section>
</For>
</div>`,

	EngHTMLTemplate: `<div class="catalog">
{{range .Categories}}
<section class="category">
  <h2>{{.Name}} ({{.Count}} items)</h2>
  <div class="items">
  {{range .Products}}
    <div class="product{{if .Featured}} featured{{end}}">
      <span class="name">{{.Name}}</span>
      <span class="price">${{.Price}}</span>
      {{if .OnSale}}<span class="sale">SALE</span>{{end}}
    </div>
  {{end}}
  </div>
</section>
{{end}}
</div>`,

	EngTextTemplate: `<div class="catalog">
{{range .Categories}}
<section class="category">
  <h2>{{.Name}} ({{.Count}} items)</h2>
  <div class="items">
  {{range .Products}}
    <div class="product{{if .Featured}} featured{{end}}">
      <span class="name">{{.Name}}</span>
      <span class="price">${{.Price}}</span>
      {{if .OnSale}}<span class="sale">SALE</span>{{end}}
    </div>
  {{end}}
  </div>
</section>
{{end}}
</div>`,

	EngPongo2: `<div class="catalog">
{% for cat in categories %}
<section class="category">
  <h2>{{ cat.name }} ({{ cat.count }} items)</h2>
  <div class="items">
  {% for product in cat.products %}
    <div class="product{% if product.featured %} featured{% endif %}">
      <span class="name">{{ product.name }}</span>
      <span class="price">${{ product.price }}</span>
      {% if product.on_sale %}<span class="sale">SALE</span>{% endif %}
    </div>
  {% endfor %}
  </div>
</section>
{% endfor %}
</div>`,

	EngJet: `<div class="catalog">
{{range _, cat := .Categories}}
<section class="category">
  <h2>{{cat.Name}} ({{cat.Count}} items)</h2>
  <div class="items">
  {{range _, product := cat.Products}}
    <div class="product{{if product.Featured}} featured{{end}}">
      <span class="name">{{product.Name}}</span>
      <span class="price">${{product.Price}}</span>
      {{if product.OnSale}}<span class="sale">SALE</span>{{end}}
    </div>
  {{end}}
  </div>
</section>
{{end}}
</div>`,

	EngLiquid: `<div class="catalog">
{% for cat in categories %}
<section class="category">
  <h2>{{ cat.name }} ({{ cat.count }} items)</h2>
  <div class="items">
  {% for product in cat.products %}
    <div class="product{% if product.featured %} featured{% endif %}">
      <span class="name">{{ product.name }}</span>
      <span class="price">${{ product.price }}</span>
      {% if product.on_sale %}<span class="sale">SALE</span>{% endif %}
    </div>
  {% endfor %}
  </div>
</section>
{% endfor %}
</div>`,
}

// --- Complex Page: full e-commerce page combining everything ---

var ComplexPageTemplates = map[string]string{
	EngGrove: `<!DOCTYPE html>
<html lang="{% lang %}">
<head>
  <meta charset="utf-8">
  <title>{% site_name %} — {% page_title %}</title>
  <meta name="description" content="{% meta_description %}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{% site_name %}</a>
      <If test={user_logged_in}>
        <span>Welcome, {% user_name %}</span>
        <a href="/cart">Cart ({% cart_count %})</a>
      <Else />
        <a href="/login">Sign In</a>
      </If>
    </nav>
  </header>
  <main>
    <h1>{% page_title %}</h1>
    <p class="lead">{% lead_text %}</p>

    <For each={categories} as="cat">
    <section class="category">
      <h2>{% cat.name %}</h2>
      <p>{% cat.description %}</p>
      <div class="products">
      <For each={cat.products} as="product">
        <div class="product"><If test={product.featured}> featured</If><If test={product.on_sale}> on-sale</If>">
          <h3>{% product.name %}</h3>
          <p>{% product.description %}</p>
          <div class="pricing">
            <span class="price">${% product.price %}</span>
            <If test={product.on_sale}><span class="original">${% product.original_price %}</span></If>
          </div>
          <span class="category-tag">{% cat.name %}</span>
          <If test={product.in_stock}>
            <button>Add to Cart</button>
          <Else />
            <button disabled>Out of Stock</button>
          </If>
          <If test={product.tags}>
          <div class="tags">
            <For each={product.tags} as="tag"><span class="tag">{% tag %}</span></For>
          </div>
          </If>
        </div>
      </For>
      </div>
    </section>
    </For>
  </main>
  <footer>
    <p>&copy; {% copyright_year %} {% site_name %}</p>
    <ul class="links">
    <For each={footer_links} as="link">
      <li><a href="{% link.url %}">{% link.label %}</a></li>
    </For>
    </ul>
  </footer>
</body>
</html>`,

	EngHTMLTemplate: `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8">
  <title>{{.SiteName}} — {{.PageTitle}}</title>
  <meta name="description" content="{{.MetaDescription}}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{.SiteName}}</a>
      {{if .UserLoggedIn}}
        <span>Welcome, {{.UserName}}</span>
        <a href="/cart">Cart ({{.CartCount}})</a>
      {{else}}
        <a href="/login">Sign In</a>
      {{end}}
    </nav>
  </header>
  <main>
    <h1>{{.PageTitle}}</h1>
    <p class="lead">{{.LeadText}}</p>

    {{range .Categories}}
    <section class="category">
      <h2>{{.Name}}</h2>
      <p>{{.Description}}</p>
      <div class="products">
      {{range .Products}}
        <div class="product{{if .Featured}} featured{{end}}{{if .OnSale}} on-sale{{end}}">
          <h3>{{.Name}}</h3>
          <p>{{.Description}}</p>
          <div class="pricing">
            <span class="price">${{.Price}}</span>
            {{if .OnSale}}<span class="original">${{.OriginalPrice}}</span>{{end}}
          </div>
          <span class="category-tag">{{$.CurrentCatName}}</span>
          {{if .InStock}}
            <button>Add to Cart</button>
          {{else}}
            <button disabled>Out of Stock</button>
          {{end}}
          {{if .Tags}}
          <div class="tags">
            {{range .Tags}}<span class="tag">{{.}}</span>{{end}}
          </div>
          {{end}}
        </div>
      {{end}}
      </div>
    </section>
    {{end}}
  </main>
  <footer>
    <p>&copy; {{.CopyrightYear}} {{.SiteName}}</p>
    <ul class="links">
    {{range .FooterLinks}}
      <li><a href="{{.URL}}">{{.Label}}</a></li>
    {{end}}
    </ul>
  </footer>
</body>
</html>`,

	EngTextTemplate: `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8">
  <title>{{.SiteName}} — {{.PageTitle}}</title>
  <meta name="description" content="{{.MetaDescription}}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{.SiteName}}</a>
      {{if .UserLoggedIn}}
        <span>Welcome, {{.UserName}}</span>
        <a href="/cart">Cart ({{.CartCount}})</a>
      {{else}}
        <a href="/login">Sign In</a>
      {{end}}
    </nav>
  </header>
  <main>
    <h1>{{.PageTitle}}</h1>
    <p class="lead">{{.LeadText}}</p>

    {{range .Categories}}
    <section class="category">
      <h2>{{.Name}}</h2>
      <p>{{.Description}}</p>
      <div class="products">
      {{range .Products}}
        <div class="product{{if .Featured}} featured{{end}}{{if .OnSale}} on-sale{{end}}">
          <h3>{{.Name}}</h3>
          <p>{{.Description}}</p>
          <div class="pricing">
            <span class="price">${{.Price}}</span>
            {{if .OnSale}}<span class="original">${{.OriginalPrice}}</span>{{end}}
          </div>
          <span class="category-tag">{{$.CurrentCatName}}</span>
          {{if .InStock}}
            <button>Add to Cart</button>
          {{else}}
            <button disabled>Out of Stock</button>
          {{end}}
          {{if .Tags}}
          <div class="tags">
            {{range .Tags}}<span class="tag">{{.}}</span>{{end}}
          </div>
          {{end}}
        </div>
      {{end}}
      </div>
    </section>
    {{end}}
  </main>
  <footer>
    <p>&copy; {{.CopyrightYear}} {{.SiteName}}</p>
    <ul class="links">
    {{range .FooterLinks}}
      <li><a href="{{.URL}}">{{.Label}}</a></li>
    {{end}}
    </ul>
  </footer>
</body>
</html>`,

	EngPongo2: `<!DOCTYPE html>
<html lang="{{ lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ site_name }} — {{ page_title }}</title>
  <meta name="description" content="{{ meta_description }}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{ site_name }}</a>
      {% if user_logged_in %}
        <span>Welcome, {{ user_name }}</span>
        <a href="/cart">Cart ({{ cart_count }})</a>
      {% else %}
        <a href="/login">Sign In</a>
      {% endif %}
    </nav>
  </header>
  <main>
    <h1>{{ page_title }}</h1>
    <p class="lead">{{ lead_text }}</p>

    {% for cat in categories %}
    <section class="category">
      <h2>{{ cat.name }}</h2>
      <p>{{ cat.description }}</p>
      <div class="products">
      {% for product in cat.products %}
        <div class="product{% if product.featured %} featured{% endif %}{% if product.on_sale %} on-sale{% endif %}">
          <h3>{{ product.name }}</h3>
          <p>{{ product.description }}</p>
          <div class="pricing">
            <span class="price">${{ product.price }}</span>
            {% if product.on_sale %}<span class="original">${{ product.original_price }}</span>{% endif %}
          </div>
          <span class="category-tag">{{ cat.name }}</span>
          {% if product.in_stock %}
            <button>Add to Cart</button>
          {% else %}
            <button disabled>Out of Stock</button>
          {% endif %}
          {% if product.tags %}
          <div class="tags">
            {% for tag in product.tags %}<span class="tag">{{ tag }}</span>{% endfor %}
          </div>
          {% endif %}
        </div>
      {% endfor %}
      </div>
    </section>
    {% endfor %}
  </main>
  <footer>
    <p>&copy; {{ copyright_year }} {{ site_name }}</p>
    <ul class="links">
    {% for link in footer_links %}
      <li><a href="{{ link.url }}">{{ link.label }}</a></li>
    {% endfor %}
    </ul>
  </footer>
</body>
</html>`,

	EngJet: `<!DOCTYPE html>
<html lang="{{ .Lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ .SiteName }} — {{ .PageTitle }}</title>
  <meta name="description" content="{{ .MetaDescription }}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{ .SiteName }}</a>
      {{if .UserLoggedIn}}
        <span>Welcome, {{ .UserName }}</span>
        <a href="/cart">Cart ({{ .CartCount }})</a>
      {{else}}
        <a href="/login">Sign In</a>
      {{end}}
    </nav>
  </header>
  <main>
    <h1>{{ .PageTitle }}</h1>
    <p class="lead">{{ .LeadText }}</p>

    {{range _, cat := .Categories}}
    <section class="category">
      <h2>{{cat.Name}}</h2>
      <p>{{cat.Description}}</p>
      <div class="products">
      {{range _, product := cat.Products}}
        <div class="product{{if product.Featured}} featured{{end}}{{if product.OnSale}} on-sale{{end}}">
          <h3>{{product.Name}}</h3>
          <p>{{product.Description}}</p>
          <div class="pricing">
            <span class="price">${{product.Price}}</span>
            {{if product.OnSale}}<span class="original">${{product.OriginalPrice}}</span>{{end}}
          </div>
          <span class="category-tag">{{cat.Name}}</span>
          {{if product.InStock}}
            <button>Add to Cart</button>
          {{else}}
            <button disabled>Out of Stock</button>
          {{end}}
          {{if product.Tags}}
          <div class="tags">
            {{range _, tag := product.Tags}}<span class="tag">{{tag}}</span>{{end}}
          </div>
          {{end}}
        </div>
      {{end}}
      </div>
    </section>
    {{end}}
  </main>
  <footer>
    <p>&copy; {{ .CopyrightYear }} {{ .SiteName }}</p>
    <ul class="links">
    {{range _, link := .FooterLinks}}
      <li><a href="{{link.URL}}">{{link.Label}}</a></li>
    {{end}}
    </ul>
  </footer>
</body>
</html>`,

	EngLiquid: `<!DOCTYPE html>
<html lang="{{ lang }}">
<head>
  <meta charset="utf-8">
  <title>{{ site_name }} — {{ page_title }}</title>
  <meta name="description" content="{{ meta_description }}">
</head>
<body>
  <header>
    <nav>
      <a href="/">{{ site_name }}</a>
      {% if user_logged_in %}
        <span>Welcome, {{ user_name }}</span>
        <a href="/cart">Cart ({{ cart_count }})</a>
      {% else %}
        <a href="/login">Sign In</a>
      {% endif %}
    </nav>
  </header>
  <main>
    <h1>{{ page_title }}</h1>
    <p class="lead">{{ lead_text }}</p>

    {% for cat in categories %}
    <section class="category">
      <h2>{{ cat.name }}</h2>
      <p>{{ cat.description }}</p>
      <div class="products">
      {% for product in cat.products %}
        <div class="product{% if product.featured %} featured{% endif %}{% if product.on_sale %} on-sale{% endif %}">
          <h3>{{ product.name }}</h3>
          <p>{{ product.description }}</p>
          <div class="pricing">
            <span class="price">${{ product.price }}</span>
            {% if product.on_sale %}<span class="original">${{ product.original_price }}</span>{% endif %}
          </div>
          <span class="category-tag">{{ cat.name }}</span>
          {% if product.in_stock %}
            <button>Add to Cart</button>
          {% else %}
            <button disabled>Out of Stock</button>
          {% endif %}
          {% if product.tags %}
          <div class="tags">
            {% for tag in product.tags %}<span class="tag">{{ tag }}</span>{% endfor %}
          </div>
          {% endif %}
        </div>
      {% endfor %}
      </div>
    </section>
    {% endfor %}
  </main>
  <footer>
    <p>&copy; {{ copyright_year }} {{ site_name }}</p>
    <ul class="links">
    {% for link in footer_links %}
      <li><a href="{{ link.url }}">{{ link.label }}</a></li>
    {% endfor %}
    </ul>
  </footer>
</body>
</html>`,
}

// --- Data types for large templates ---

type Product struct {
	Name          string
	Description   string
	Price         string
	OriginalPrice string
	Category      string
	InStock       bool
	OnSale        bool
	Featured      bool
	Tags          []string
}

type Category struct {
	Name        string
	Description string
	Count       int
	Products    []Product
}

type FooterLink struct {
	URL   string
	Label string
}

type LargePageData struct {
	Lang            string
	SiteName        string
	PageTitle       string
	MetaDescription string
	MetaAuthor      string
	CanonicalURL    string
	Tagline         string
	UserName        string
	UserRole        string
	LeadText        string
	BodyText        string
	Category        string
	PublishedDate   string
	ReadingTime     int
	CopyrightYear   int
	FooterText      string
}

type LargeLoopData struct {
	Products []Product
}

type NestedLoopData struct {
	Categories []Category
}

type ComplexPageData struct {
	Lang            string
	SiteName        string
	PageTitle       string
	MetaDescription string
	UserLoggedIn    bool
	UserName        string
	CartCount       int
	LeadText        string
	Categories      []Category
	CurrentCatName  string
	CopyrightYear   int
	FooterLinks     []FooterLink
}

// --- Data constructors ---

func NewLargePageData() map[string]any {
	m := map[string]any{
		"lang":             "en",
		"site_name":        "GroveShop",
		"page_title":       "Welcome to Our Store",
		"meta_description": "The best online store for everything you need. Shop now for great deals.",
		"meta_author":      "GroveShop Team",
		"canonical_url":    "https://groveshop.example.com/",
		"tagline":          "Your one-stop shop for quality goods",
		"user_name":        "Alice",
		"user_role":        "Premium Member",
		"lead_text":        "Discover our curated collection of premium products, handpicked for quality and value.",
		"body_text":        "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
		"category":         "Featured",
		"published_date":   "2026-03-15",
		"reading_time":     5,
		"copyright_year":   2026,
		"footer_text":      "Built with Grove template engine.",
	}
	m["_struct"] = LargePageData{
		Lang:            "en",
		SiteName:        "GroveShop",
		PageTitle:       "Welcome to Our Store",
		MetaDescription: "The best online store for everything you need. Shop now for great deals.",
		MetaAuthor:      "GroveShop Team",
		CanonicalURL:    "https://groveshop.example.com/",
		Tagline:         "Your one-stop shop for quality goods",
		UserName:        "Alice",
		UserRole:        "Premium Member",
		LeadText:        "Discover our curated collection of premium products, handpicked for quality and value.",
		BodyText:        "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
		Category:        "Featured",
		PublishedDate:   "2026-03-15",
		ReadingTime:     5,
		CopyrightYear:   2026,
		FooterText:      "Built with Grove template engine.",
	}
	return m
}

func newProducts(count int) []Product {
	names := []string{"Widget Pro", "Gadget X", "Super Thingamajig", "Mega Doohickey", "Ultra Whatchamacallit",
		"Premium Gizmo", "Deluxe Contraption", "Elite Apparatus", "Pro Device", "Max Instrument"}
	categories := []string{"Electronics", "Home & Garden", "Sports", "Books", "Clothing"}
	tags := [][]string{
		{"new", "popular", "sale"},
		{"bestseller", "premium"},
		{"eco-friendly", "handmade"},
		{"limited", "exclusive"},
		nil, // some products have no tags
	}

	products := make([]Product, count)
	for i := range products {
		ni := i % len(names)
		ci := i % len(categories)
		ti := i % len(tags)
		products[i] = Product{
			Name:          fmt.Sprintf("%s #%d", names[ni], i+1),
			Description:   fmt.Sprintf("High-quality %s for everyday use. Model %d with enhanced features.", names[ni], i+1),
			Price:         fmt.Sprintf("%.2f", 9.99+float64(i)*3.50),
			OriginalPrice: fmt.Sprintf("%.2f", 14.99+float64(i)*3.50),
			Category:      categories[ci],
			InStock:       i%4 != 0,
			OnSale:        i%3 == 0,
			Featured:      i%5 == 0,
			Tags:          tags[ti],
		}
	}
	return products
}

func NewLargeLoopData() map[string]any {
	products := newProducts(100)
	// Map version for Grove/Pongo2
	pmaps := make([]map[string]any, len(products))
	for i, p := range products {
		pmaps[i] = map[string]any{
			"name":           p.Name,
			"description":    p.Description,
			"price":          p.Price,
			"original_price": p.OriginalPrice,
			"category":       p.Category,
			"in_stock":       p.InStock,
			"on_sale":        p.OnSale,
			"featured":       p.Featured,
			"tags":           p.Tags,
		}
	}
	m := map[string]any{"products": pmaps}
	m["_struct"] = LargeLoopData{Products: products}
	return m
}

func newCategories() []Category {
	catNames := []string{"Electronics", "Home & Garden", "Sports & Outdoors", "Books & Media",
		"Clothing & Accessories", "Health & Beauty", "Toys & Games", "Automotive",
		"Food & Grocery", "Office Supplies"}
	catDescs := []string{
		"Latest gadgets and electronic devices",
		"Everything for your home and garden",
		"Gear up for your favorite sports",
		"Books, movies, music, and more",
		"Fashion for every occasion",
		"Look and feel your best",
		"Fun for all ages",
		"Parts, accessories, and tools",
		"Fresh food and pantry essentials",
		"Supplies for your workspace",
	}
	cats := make([]Category, len(catNames))
	for i, name := range catNames {
		prods := newProducts(10)
		for j := range prods {
			prods[j].Category = name
		}
		cats[i] = Category{
			Name:        name,
			Description: catDescs[i],
			Count:       len(prods),
			Products:    prods,
		}
	}
	return cats
}

func NewNestedLoopData() map[string]any {
	cats := newCategories()
	// Map version
	cmaps := make([]map[string]any, len(cats))
	for i, c := range cats {
		pmaps := make([]map[string]any, len(c.Products))
		for j, p := range c.Products {
			pmaps[j] = map[string]any{
				"name":           p.Name,
				"description":    p.Description,
				"price":          p.Price,
				"original_price": p.OriginalPrice,
				"category":       p.Category,
				"in_stock":       p.InStock,
				"on_sale":        p.OnSale,
				"featured":       p.Featured,
				"tags":           p.Tags,
			}
		}
		cmaps[i] = map[string]any{
			"name":        c.Name,
			"description": c.Description,
			"count":       c.Count,
			"products":    pmaps,
		}
	}
	m := map[string]any{"categories": cmaps}
	m["_struct"] = NestedLoopData{Categories: cats}
	return m
}

func NewComplexPageData() map[string]any {
	cats := newCategories()
	links := []FooterLink{
		{URL: "/about", Label: "About Us"},
		{URL: "/contact", Label: "Contact"},
		{URL: "/privacy", Label: "Privacy Policy"},
		{URL: "/terms", Label: "Terms of Service"},
		{URL: "/faq", Label: "FAQ"},
	}

	// Map version
	cmaps := make([]map[string]any, len(cats))
	for i, c := range cats {
		pmaps := make([]map[string]any, len(c.Products))
		for j, p := range c.Products {
			pmaps[j] = map[string]any{
				"name":           p.Name,
				"description":    p.Description,
				"price":          p.Price,
				"original_price": p.OriginalPrice,
				"category":       p.Category,
				"in_stock":       p.InStock,
				"on_sale":        p.OnSale,
				"featured":       p.Featured,
				"tags":           p.Tags,
			}
		}
		cmaps[i] = map[string]any{
			"name":        c.Name,
			"description": c.Description,
			"count":       c.Count,
			"products":    pmaps,
		}
	}

	lmaps := make([]map[string]any, len(links))
	for i, l := range links {
		lmaps[i] = map[string]any{"url": l.URL, "label": l.Label}
	}

	m := map[string]any{
		"lang":             "en",
		"site_name":        "GroveShop",
		"page_title":       "Shop All Categories",
		"meta_description": "Browse our complete catalog across all categories.",
		"user_logged_in":   true,
		"user_name":        "Alice",
		"cart_count":       3,
		"lead_text":        "Explore our full range of products across every category.",
		"categories":       cmaps,
		"copyright_year":   2026,
		"footer_links":     lmaps,
	}
	m["_struct"] = ComplexPageData{
		Lang:            "en",
		SiteName:        "GroveShop",
		PageTitle:       "Shop All Categories",
		MetaDescription: "Browse our complete catalog across all categories.",
		UserLoggedIn:    true,
		UserName:        "Alice",
		CartCount:       3,
		LeadText:        "Explore our full range of products across every category.",
		Categories:      cats,
		CopyrightYear:   2026,
		FooterLinks:     links,
	}
	return m
}
