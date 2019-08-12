/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var testAccount = &do.Account{
	Account: &godo.Account{
		DropletLimit:  10,
		Email:         "user@example.com",
		UUID:          "1234",
		EmailVerified: true,
	},
}

func TestAccountCommand(t *testing.T) {
	acctCmd := Account()
	assert.NotNil(t, acctCmd)
	assertCommandNames(t, acctCmd, "get", "ratelimit")
}

func TestAccountGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.EXPECT().Get().Return(testAccount, nil)

		err := RunAccountGet(config)
		assert.NoError(t, err)
	})

	if !testing.Short() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/account":
				w.Write([]byte(`{
					"account": {
						"droplet_limit": 25,
						"floating_ip_limit": 5,
						"email": "sammy@digitalocean.com",
						"uuid": "b6fr89dbf6d9156cace5f3c78dc9851d957381ef",
						"email_verified": true,
						"status": "active",
						"status_message": ""
						}
					}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		t.Run("it returns the details of my account", func(t *testing.T) {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"account",
				"get",
			)

			output, err := cmd.CombinedOutput()
			assert.NoError(t, err)

			exitCode := cmd.ProcessState.ExitCode()
			assert.Equal(t, 0, exitCode, "exit code should be zero")

			assert.Equal(t, strings.TrimSpace(accountOutput), strings.TrimSpace(string(output)))
		})
	}
}

const accountOutput string = `
Email                     Droplet Limit    Email Verified    UUID                                        Status
sammy@digitalocean.com    25               true              b6fr89dbf6d9156cace5f3c78dc9851d957381ef    active
`

const ratelimitOutput string = `
Limit    Remaining    Reset
200      199          %s
`
