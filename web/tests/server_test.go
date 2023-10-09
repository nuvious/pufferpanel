package tests

import (
	"encoding/json"
	"github.com/pufferpanel/pufferpanel/v3/messages"
	"github.com/pufferpanel/pufferpanel/v3/servers"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestServers(t *testing.T) {
	t.Run("CreateServer", func(t *testing.T) {
		session, err := createSessionAdmin()
		if !assert.NoError(t, err) {
			return
		}

		response := CallAPIRaw("PUT", "/api/servers/testserver", CreateServerData, session)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("GetStats", func(t *testing.T) {
		session, err := createSessionAdmin()
		if !assert.NoError(t, err) {
			return
		}

		response := CallAPI("GET", "/api/servers/testserver/stats", nil, session)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("SendStatsForServers", func(t *testing.T) {
		servers.SendStatsForServers()
	})

	t.Run("GetEmptyFiles", func(t *testing.T) {
		session, err := createSessionAdmin()
		if !assert.NoError(t, err) {
			return
		}

		response := CallAPI("GET", "/api/servers/testserver/file/", nil, session)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("InstallServer", func(t *testing.T) {
		session, err := createSessionAdmin()
		if !assert.NoError(t, err) {
			return
		}

		response := CallAPI("POST", "/api/servers/testserver/install", nil, session)
		if !assert.Equal(t, http.StatusAccepted, response.Code) {
			return
		}

		time.Sleep(100 * time.Millisecond)

		//we expect it to take more than 100ms, so ensure there is an install occurring
		response = CallAPI("GET", "/api/servers/testserver/status", nil, session)
		assert.Equal(t, http.StatusOK, response.Code)
		var status messages.Status
		err = json.NewDecoder(response.Body).Decode(&status)
		if !assert.NoError(t, err) {
			return
		}
		if !assert.True(t, status.Installing) {
			return
		}

		//now we wait for the install to finish
		timeout := 60
		counter := 0
		for counter < timeout {
			time.Sleep(time.Second)
			response = CallAPI("GET", "/api/servers/testserver/status", nil, session)
			assert.Equal(t, http.StatusOK, response.Code)
			var status messages.Status
			err = json.NewDecoder(response.Body).Decode(&status)
			if !assert.NoError(t, err) {
				return
			}
			if status.Installing {
				counter++
			} else {
				break
			}
		}
		if counter >= timeout {
			assert.Fail(t, "Server took too long to install, assuming test failed")
		}
	})

	t.Run("StartServer", func(t *testing.T) {
		session, err := createSessionAdmin()
		if !assert.NoError(t, err) {
			return
		}

		response := CallAPI("POST", "/api/servers/testserver/start", nil, session)
		assert.Equal(t, http.StatusAccepted, response.Code)

		time.Sleep(1000 * time.Millisecond)

		//we expect it to take more than 1 second, so ensure there is a started server
		response = CallAPI("GET", "/api/servers/testserver/status", nil, session)
		assert.Equal(t, http.StatusOK, response.Code)
		var status messages.Status
		err = json.NewDecoder(response.Body).Decode(&status)
		if !assert.NoError(t, err) {
			return
		}
		if !assert.True(t, status.Running) {
			return
		}
	})

}
