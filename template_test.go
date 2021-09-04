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
	// render text
	html, err := ht.Render(context.Background(), text, &Data{
		ID:   1,
		Name: "tree.xie",
	})
	assert.Nil(err)
	assert.Equal("<p>1<span>tree.xie</span></p>", html)

	// render file
	f, err := ioutil.TempFile("", "")
	assert.Nil(err)
	filename := f.Name()
	defer os.Remove(filename)
	_, err = f.WriteString(text)
	assert.Nil(err)
	err = f.Close()
	assert.Nil(err)
	html, err = ht.RenderFile(context.Background(), filename, &Data{
		ID:   2,
		Name: "tree",
	})
	assert.Nil(err)
	assert.Equal("<p>2<span>tree</span></p>", html)
}