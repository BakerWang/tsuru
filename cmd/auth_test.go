package cmd

import (
	"bytes"
	"errors"
	"github.com/timeredbull/tsuru/fs/testing"
	"io"
	. "launchpad.net/gocheck"
	"net/http"
)

func (s *S) TestLogin(c *C) {
	fsystem = &testing.RecordingFs{FileContent: "old-token"}
	defer func() {
		fsystem = nil
	}()
	expected := "Successfully logged!\n"
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: `{"token": "sometoken"}`, status: http.StatusOK}})
	command := login{reader: &fakeReader{outputs: []string{"chico"}}}
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	token, err := readToken()
	c.Assert(err, IsNil)
	c.Assert(token, Equals, "sometoken")
}

func (s *S) TestLoginShouldNotDependOnTsuruTokenFile(c *C) {
	fsystem = &testing.FailureFs{}
	defer func() {
		fsystem = nil
	}()
	expected := "Successfully logged!\n"
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: `{"token":"anothertoken"}`, status: http.StatusOK}})
	command := login{reader: &fakeReader{outputs: []string{"bar123"}}}
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestLoginShouldReturnErrorIfThePasswordIsNotGiven(c *C) {
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	command := login{reader: &failingReader{msg: "You must provide the password!"}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestLoginPreader(c *C) {
	reader := fakeReader{outputs: []string{"123", "456"}}
	login := login{}
	login.reader = &reader
	c.Assert(login.preader(), DeepEquals, &reader)
	login.reader = nil
	c.Assert(login.preader(), DeepEquals, stdinPasswordReader{})
}

func (s *S) TestLogout(c *C) {
	rfs := &testing.RecordingFs{}
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	expected := "Successfully logout!\n"
	context := Context{[]string{}, []string{}, manager.Stdout, manager.Stderr}
	command := logout{}
	err := command.Run(&context, nil)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	tokenPath, err := joinWithUserDir(".tsuru_token")
	c.Assert(err, IsNil)
	c.Assert(rfs.HasAction("remove "+tokenPath), Equals, true)
}

func (s *S) TestLogoutWhenNotLoggedIn(c *C) {
	fsystem = &testing.FailureFs{}
	defer func() {
		fsystem = nil
	}()
	context := Context{[]string{}, []string{}, manager.Stdout, manager.Stderr}
	command := logout{}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "You're not logged in!")
}

func (s *S) TestTeamAddUser(c *C) {
	expected := `User "andorito" was added to the "cobrateam" team` + "\n"
	context := Context{[]string{}, []string{"cobrateam", "andorito"}, manager.Stdout, manager.Stderr}
	command := teamUserAdd{}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusOK}})
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestTeamAddUserInfo(c *C) {
	expected := &Info{
		Name:    "team-user-add",
		Usage:   "team-user-add <teamname> <useremail>",
		Desc:    "adds a user to a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserAdd{}).Info(), DeepEquals, expected)
}

func (s *S) TestTeamRemoveUser(c *C) {
	expected := `User "andorito" was removed from the "cobrateam" team` + "\n"
	context := Context{[]string{}, []string{"cobrateam", "andorito"}, manager.Stdout, manager.Stderr}
	command := teamUserRemove{}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusOK}})
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestTeamRemoveUserInfo(c *C) {
	expected := &Info{
		Name:    "team-user-remove",
		Usage:   "team-user-remove <teamname> <useremail>",
		Desc:    "removes a user from a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserRemove{}).Info(), DeepEquals, expected)
}

func (s *S) TestTeamCreate(c *C) {
	expected := `Team "core" successfully created!` + "\n"
	context := Context{[]string{}, []string{"core"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}})
	command := teamCreate{}
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestTeamCreateInfo(c *C) {
	expected := &Info{
		Name:    "team-create",
		Usage:   "team-create <teamname>",
		Desc:    "creates a new team.",
		MinArgs: 1,
	}
	c.Assert((&teamCreate{}).Info(), DeepEquals, expected)
}

func (s *S) TestTeamListRun(c *C) {
	var called bool
	trans := &conditionalTransport{
		transport{
			msg:    `[{"name":"timeredbull"},{"name":"cobrateam"}]`,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && req.URL.Path == "/teams"
		},
	}
	expected := `Teams:

  - timeredbull
  - cobrateam
`
	client := NewClient(&http.Client{Transport: trans})
	err := (&teamList{}).Run(&Context{[]string{}, []string{}, manager.Stdout, manager.Stderr}, client)
	c.Assert(err, IsNil)
	c.Assert(called, Equals, true)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestTeamListRunWithNoContent(c *C) {
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusNoContent}})
	err := (&teamList{}).Run(&Context{[]string{}, []string{}, manager.Stdout, manager.Stderr}, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, "")
}

func (s *S) TestTeamListInfo(c *C) {
	expected := &Info{
		Name:    "team-list",
		Usage:   "team-list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
	c.Assert((&teamList{}).Info(), DeepEquals, expected)
}

func (s *S) TestTeamListIsACommand(c *C) {
	var command Command
	c.Assert(&teamList{}, Implements, &command)
}

func (s *S) TeamTeamListIsAnInfoer(c *C) {
	var infoer Infoer
	c.Assert(&teamList{}, Implements, &infoer)
}

func (s *S) TestUserCreateShouldNotDependOnTsuruTokenFile(c *C) {
	fsystem = &testing.FailureFs{}
	defer func() {
		fsystem = nil
	}()
	expected := `User "foo@foo.com" successfully created!` + "\n"
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}})
	command := userCreate{reader: &fakeReader{outputs: []string{"foo123"}}}
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestUserCreateReturnErrorIfPasswordsDontMatch(c *C) {
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}})
	command := userCreate{reader: &fakeReader{outputs: []string{"foo123", "foo1234"}}}
	err := command.Run(&context, client)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Passwords didn't match.$")
}

func (s *S) TestUserCreate(c *C) {
	expected := `User "foo@foo.com" successfully created!` + "\n"
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}})
	command := userCreate{reader: &fakeReader{outputs: []string{"foo123"}}}
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
}

func (s *S) TestUserCreateShouldReturnErrorIfThePasswordIsNotGiven(c *C) {
	context := Context{[]string{}, []string{"foo@foo.com"}, manager.Stdout, manager.Stderr}
	command := userCreate{reader: &failingReader{msg: "You must provide the password!"}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestUserCreateInfo(c *C) {
	expected := &Info{
		Name:    "user-create",
		Usage:   "user-create <email>",
		Desc:    "creates a user.",
		MinArgs: 1,
	}
	c.Assert((&userCreate{}).Info(), DeepEquals, expected)
}

func (s *S) TestUserCreatePreader(c *C) {
	reader := fakeReader{outputs: []string{"123", "456"}}
	create := userCreate{}
	create.reader = &reader
	c.Assert(create.preader(), DeepEquals, &reader)
	create.reader = nil
	c.Assert(create.preader(), DeepEquals, stdinPasswordReader{})
}

type fakeReader struct {
	reads   int
	outputs []string
}

func (r *fakeReader) readPassword(out io.Writer, msg string) (string, error) {
	output := r.outputs[r.reads%len(r.outputs)]
	r.reads++
	return output, nil
}

type failingReader struct {
	msg string
}

func (r *failingReader) readPassword(out io.Writer, msg string) (string, error) {
	return "", errors.New(r.msg)
}
