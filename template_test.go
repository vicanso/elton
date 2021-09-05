package elton

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLTemplate(t *testing.T) {
	assert := assert.New(t)
	ht := HTMLTemplate{}

	type Data struct {
		ID   int
		Name string
	}

	text := "<p>{{.ID}}<span>{{.Name}}</span></p>"

	t.Run("render text", func(t *testing.T) {
		// render text
		html, err := ht.Render(context.Background(), text, &Data{
			ID:   1,
			Name: "tree.xie",
		})
		assert.Nil(err)
		assert.Equal("<p>1<span>tree.xie</span></p>", html)
	})

	t.Run("render file", func(t *testing.T) {
		// render file
		f, err := ioutil.TempFile("", "")
		assert.Nil(err)
		filename := f.Name()
		defer os.Remove(filename)
		_, err = f.WriteString(text)
		assert.Nil(err)
		err = f.Close()
		assert.Nil(err)
		html, err := ht.RenderFile(context.Background(), filename, &Data{
			ID:   2,
			Name: "tree",
		})
		assert.Nil(err)
		assert.Equal("<p>2<span>tree</span></p>", html)
	})
}

func TestTemplates(t *testing.T) {
	assert := assert.New(t)

	assert.NotNil(GetParser("html"))
	assert.NotNil(GetParser("tmpl"))
}
