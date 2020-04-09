package elton

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTree(t *testing.T) {
	assert := assert.New(t)

	hStub := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hIndex := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hFavicon := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleList := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleNear := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleShow := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleShowRelated := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleShowOpts := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleSlug := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hArticleByUser := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hUserList := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hUserShow := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hAdminCatchall := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hAdminAppShow := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hAdminAppShowCatchall := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hUserProfile := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hUserSuper := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hUserAll := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hHubView1 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hHubView2 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hHubView3 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}

	tr := &node{}

	tr.InsertRoute(mGET, "/", hIndex)
	tr.InsertRoute(mGET, "/favicon.ico", hFavicon)

	tr.InsertRoute(mGET, "/pages/*", hStub)

	tr.InsertRoute(mGET, "/article", hArticleList)
	tr.InsertRoute(mGET, "/article/", hArticleList)

	tr.InsertRoute(mGET, "/article/near", hArticleNear)
	tr.InsertRoute(mGET, "/article/{id}", hStub)
	tr.InsertRoute(mGET, "/article/{id}", hArticleShow)
	tr.InsertRoute(mGET, "/article/{id}", hArticleShow) // duplicate will have no effect
	tr.InsertRoute(mGET, "/article/@{user}", hArticleByUser)

	tr.InsertRoute(mGET, "/article/{sup}/{opts}", hArticleShowOpts)
	tr.InsertRoute(mGET, "/article/{id}/{opts}", hArticleShowOpts) // overwrite above route, latest wins

	tr.InsertRoute(mGET, "/article/{iffd}/edit", hStub)
	tr.InsertRoute(mGET, "/article/{id}//related", hArticleShowRelated)
	tr.InsertRoute(mGET, "/article/slug/{month}/-/{day}/{year}", hArticleSlug)

	tr.InsertRoute(mGET, "/admin/user", hUserList)
	tr.InsertRoute(mGET, "/admin/user/", hStub) // will get replaced by next route
	tr.InsertRoute(mGET, "/admin/user/", hUserList)

	tr.InsertRoute(mGET, "/admin/user//{id}", hUserShow)
	tr.InsertRoute(mGET, "/admin/user/{id}", hUserShow)

	tr.InsertRoute(mGET, "/admin/apps/{id}", hAdminAppShow)
	tr.InsertRoute(mGET, "/admin/apps/{id}/*", hAdminAppShowCatchall)

	tr.InsertRoute(mGET, "/admin/*", hStub) // catchall segment will get replaced by next route
	tr.InsertRoute(mGET, "/admin/*", hAdminCatchall)

	tr.InsertRoute(mGET, "/users/{userID}/profile", hUserProfile)
	tr.InsertRoute(mGET, "/users/super/*", hUserSuper)
	tr.InsertRoute(mGET, "/users/*", hUserAll)

	tr.InsertRoute(mGET, "/hubs/{hubID}/view", hHubView1)
	tr.InsertRoute(mGET, "/hubs/{hubID}/view/*", hHubView2)
	tr.InsertRoute(mGET, "/hubs/{hubID}/users", hHubView3)

	tests := []struct {
		r string          // input request path
		h EndpointHandler // output matched handler
		k []string        // output param keys
		v []string        // output param values
	}{
		{r: "/", h: hIndex, k: nil, v: nil},
		{r: "/favicon.ico", h: hFavicon, k: nil, v: nil},

		{r: "/pages", h: nil, k: nil, v: nil},
		{r: "/pages/", h: hStub, k: []string{"*"}, v: []string{""}},
		{r: "/pages/yes", h: hStub, k: []string{"*"}, v: []string{"yes"}},

		{r: "/article", h: hArticleList, k: nil, v: nil},
		{r: "/article/", h: hArticleList, k: nil, v: nil},
		{r: "/article/near", h: hArticleNear, k: nil, v: nil},
		{r: "/article/neard", h: hArticleShow, k: []string{"id"}, v: []string{"neard"}},
		{r: "/article/123", h: hArticleShow, k: []string{"id"}, v: []string{"123"}},
		{r: "/article/123/456", h: hArticleShowOpts, k: []string{"id", "opts"}, v: []string{"123", "456"}},
		{r: "/article/@peter", h: hArticleByUser, k: []string{"user"}, v: []string{"peter"}},
		{r: "/article/22//related", h: hArticleShowRelated, k: []string{"id"}, v: []string{"22"}},
		{r: "/article/111/edit", h: hStub, k: []string{"iffd"}, v: []string{"111"}},
		{r: "/article/slug/sept/-/4/2015", h: hArticleSlug, k: []string{"month", "day", "year"}, v: []string{"sept", "4", "2015"}},
		{r: "/article/:id", h: hArticleShow, k: []string{"id"}, v: []string{":id"}},

		{r: "/admin/user", h: hUserList, k: nil, v: nil},
		{r: "/admin/user/", h: hUserList, k: nil, v: nil},
		{r: "/admin/user/1", h: hUserShow, k: []string{"id"}, v: []string{"1"}},
		{r: "/admin/user//1", h: hUserShow, k: []string{"id"}, v: []string{"1"}},
		{r: "/admin/hi", h: hAdminCatchall, k: []string{"*"}, v: []string{"hi"}},
		{r: "/admin/lots/of/:fun", h: hAdminCatchall, k: []string{"*"}, v: []string{"lots/of/:fun"}},
		{r: "/admin/apps/333", h: hAdminAppShow, k: []string{"id"}, v: []string{"333"}},
		{r: "/admin/apps/333/woot", h: hAdminAppShowCatchall, k: []string{"id", "*"}, v: []string{"333", "woot"}},

		{r: "/hubs/123/view", h: hHubView1, k: []string{"hubID"}, v: []string{"123"}},
		{r: "/hubs/123/view/index.html", h: hHubView2, k: []string{"hubID", "*"}, v: []string{"123", "index.html"}},
		{r: "/hubs/123/users", h: hHubView3, k: []string{"hubID"}, v: []string{"123"}},

		{r: "/users/123/profile", h: hUserProfile, k: []string{"userID"}, v: []string{"123"}},
		{r: "/users/super/123/okay/yes", h: hUserSuper, k: []string{"*"}, v: []string{"123/okay/yes"}},
		{r: "/users/123/okay/yes", h: hUserAll, k: []string{"*"}, v: []string{"123/okay/yes"}},
	}

	for _, tt := range tests {

		handler, params := tr.FindRoute(mGET, tt.r)

		assert.Equal(fmt.Sprintf("%v", tt.h), fmt.Sprintf("%v", handler))
		assert.Equal(tt.k, params.Keys)
		assert.Equal(tt.v, params.Values)
	}

	// test method not allowed
	params := &RouteParams{
		Keys:   nil,
		Values: nil,
	}
	node := tr.findRoute(mPOST, "/", params)
	assert.Nil(node)
	assert.True(params.methodNotAllowed)
}

func TestTreeMoar(t *testing.T) {
	assert := assert.New(t)
	hStub := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub1 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub2 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub3 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub4 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub5 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub6 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub7 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub8 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub9 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub10 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub11 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub12 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub13 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub14 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub15 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub16 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}

	// TODO: panic if we see {id}{x} because we're missing a delimiter, its not possible.
	// also {:id}* is not possible.

	tr := &node{}

	tr.InsertRoute(mGET, "/articlefun", hStub5)
	tr.InsertRoute(mGET, "/articles/{id}", hStub)
	tr.InsertRoute(mDELETE, "/articles/{slug}", hStub8)
	tr.InsertRoute(mGET, "/articles/search", hStub1)
	tr.InsertRoute(mGET, "/articles/{id}:delete", hStub8)
	tr.InsertRoute(mGET, "/articles/{iidd}!sup", hStub4)
	tr.InsertRoute(mGET, "/articles/{id}:{op}", hStub3)
	tr.InsertRoute(mGET, "/articles/{id}:{op}", hStub2)                              // this route sets a new handler for the above route
	tr.InsertRoute(mGET, "/articles/{slug:^[a-z]+}/posts", hStub)                    // up to tail '/' will only match if contents match the rex
	tr.InsertRoute(mGET, "/articles/{id}/posts/{pid}", hStub6)                       // /articles/123/posts/1
	tr.InsertRoute(mGET, "/articles/{id}/posts/{month}/{day}/{year}/{slug}", hStub7) // /articles/123/posts/09/04/1984/juice
	tr.InsertRoute(mGET, "/articles/{id}.json", hStub10)
	tr.InsertRoute(mGET, "/articles/{id}/data.json", hStub11)
	tr.InsertRoute(mGET, "/articles/files/{file}.{ext}", hStub12)
	tr.InsertRoute(mPUT, "/articles/me", hStub13)

	// TODO: make a separate test case for this one..
	// tr.InsertRoute(mGET, "/articles/{id}/{id}", hStub1)                              // panic expected, we're duplicating param keys

	tr.InsertRoute(mGET, "/pages/*", hStub)
	tr.InsertRoute(mGET, "/pages/*", hStub9)

	tr.InsertRoute(mGET, "/users/{id}", hStub14)
	tr.InsertRoute(mGET, "/users/{id}/settings/{key}", hStub15)
	tr.InsertRoute(mGET, "/users/{id}/settings/*", hStub16)

	tests := []struct {
		m methodTyp       // input request http method
		r string          // input request path
		h EndpointHandler // output matched handler
		k []string        // output param keys
		v []string        // output param values
	}{
		{m: mGET, r: "/articles/search", h: hStub1, k: nil, v: nil},
		{m: mGET, r: "/articlefun", h: hStub5, k: nil, v: nil},
		{m: mGET, r: "/articles/123", h: hStub, k: []string{"id"}, v: []string{"123"}},
		{m: mDELETE, r: "/articles/123mm", h: hStub8, k: []string{"slug"}, v: []string{"123mm"}},
		{m: mGET, r: "/articles/789:delete", h: hStub8, k: []string{"id"}, v: []string{"789"}},
		{m: mGET, r: "/articles/789!sup", h: hStub4, k: []string{"iidd"}, v: []string{"789"}},
		{m: mGET, r: "/articles/123:sync", h: hStub2, k: []string{"id", "op"}, v: []string{"123", "sync"}},
		{m: mGET, r: "/articles/456/posts/1", h: hStub6, k: []string{"id", "pid"}, v: []string{"456", "1"}},
		{m: mGET, r: "/articles/456/posts/09/04/1984/juice", h: hStub7, k: []string{"id", "month", "day", "year", "slug"}, v: []string{"456", "09", "04", "1984", "juice"}},
		{m: mGET, r: "/articles/456.json", h: hStub10, k: []string{"id"}, v: []string{"456"}},
		{m: mGET, r: "/articles/456/data.json", h: hStub11, k: []string{"id"}, v: []string{"456"}},

		{m: mGET, r: "/articles/files/file.zip", h: hStub12, k: []string{"file", "ext"}, v: []string{"file", "zip"}},
		{m: mGET, r: "/articles/files/photos.tar.gz", h: hStub12, k: []string{"file", "ext"}, v: []string{"photos", "tar.gz"}},
		{m: mGET, r: "/articles/files/photos.tar.gz", h: hStub12, k: []string{"file", "ext"}, v: []string{"photos", "tar.gz"}},

		{m: mPUT, r: "/articles/me", h: hStub13, k: nil, v: nil},
		{m: mGET, r: "/articles/me", h: hStub, k: []string{"id"}, v: []string{"me"}},
		{m: mGET, r: "/pages", h: nil, k: nil, v: nil},
		{m: mGET, r: "/pages/", h: hStub9, k: []string{"*"}, v: []string{""}},
		{m: mGET, r: "/pages/yes", h: hStub9, k: []string{"*"}, v: []string{"yes"}},

		{m: mGET, r: "/users/1", h: hStub14, k: []string{"id"}, v: []string{"1"}},
		{m: mGET, r: "/users/", h: nil, k: nil, v: nil},
		{m: mGET, r: "/users/2/settings/password", h: hStub15, k: []string{"id", "key"}, v: []string{"2", "password"}},
		{m: mGET, r: "/users/2/settings/", h: hStub16, k: []string{"id", "*"}, v: []string{"2", ""}},
	}

	for _, tt := range tests {

		handler, params := tr.FindRoute(tt.m, tt.r)

		assert.Equal(fmt.Sprintf("%v", tt.h), fmt.Sprintf("%v", handler))
		assert.Equal(tt.k, params.Keys)
		assert.Equal(tt.v, params.Values)
	}
}

func TestTreeRegexp(t *testing.T) {
	assert := assert.New(t)
	hStub1 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub2 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub3 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub4 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub5 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub6 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}
	hStub7 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}

	tr := &node{}
	tr.InsertRoute(mGET, "/articles/{rid:^[0-9]{5,6}}", hStub7)
	tr.InsertRoute(mGET, "/articles/{zid:^0[0-9]+}", hStub3)
	tr.InsertRoute(mGET, "/articles/{name:^@[a-z]+}/posts", hStub4)
	tr.InsertRoute(mGET, "/articles/{op:^[0-9]+}/run", hStub5)
	tr.InsertRoute(mGET, "/articles/{id:^[0-9]+}", hStub1)
	tr.InsertRoute(mGET, "/articles/{id:^[1-9]+}-{aux}", hStub6)
	tr.InsertRoute(mGET, "/articles/{slug}", hStub2)

	tests := []struct {
		r string          // input request path
		h EndpointHandler // output matched handler
		k []string        // output param keys
		v []string        // output param values
	}{
		{r: "/articles", h: nil, k: nil, v: nil},
		{r: "/articles/12345", h: hStub7, k: []string{"rid"}, v: []string{"12345"}},
		{r: "/articles/123", h: hStub1, k: []string{"id"}, v: []string{"123"}},
		{r: "/articles/how-to-build-a-router", h: hStub2, k: []string{"slug"}, v: []string{"how-to-build-a-router"}},
		{r: "/articles/0456", h: hStub3, k: []string{"zid"}, v: []string{"0456"}},
		{r: "/articles/@pk/posts", h: hStub4, k: []string{"name"}, v: []string{"@pk"}},
		{r: "/articles/1/run", h: hStub5, k: []string{"op"}, v: []string{"1"}},
		{r: "/articles/1122", h: hStub1, k: []string{"id"}, v: []string{"1122"}},
		{r: "/articles/1122-yes", h: hStub6, k: []string{"id", "aux"}, v: []string{"1122", "yes"}},
	}

	for _, tt := range tests {
		handler, params := tr.FindRoute(mGET, tt.r)

		assert.Equal(fmt.Sprintf("%v", tt.h), fmt.Sprintf("%v", handler))
		assert.Equal(tt.k, params.Keys)
		assert.Equal(tt.v, params.Values)
	}
}

func TestTreeRegexMatchWholeParam(t *testing.T) {
	assert := assert.New(t)
	hStub1 := func(w http.ResponseWriter, r *http.Request, params *RouteParams) {}

	tr := &node{}
	tr.InsertRoute(mGET, "/{id:[0-9]+}", hStub1)

	tests := []struct {
		url             string
		expectedHandler EndpointHandler
	}{
		{url: "/13", expectedHandler: hStub1},
		{url: "/a13", expectedHandler: nil},
		{url: "/13.jpg", expectedHandler: nil},
		{url: "/a13.jpg", expectedHandler: nil},
	}

	for _, tc := range tests {
		handler, _ := tr.FindRoute(mGET, tc.url)
		assert.Equal(fmt.Sprintf("%v", tc.expectedHandler), fmt.Sprintf("%v", handler))
	}
}
