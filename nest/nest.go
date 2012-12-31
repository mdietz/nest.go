package nest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Nest struct {
	username string
	password string

	access_token  string
	userid        string
	transport_url string

	client         *http.Client
	status_request *http.Request
}

type LoginResponse struct {
	Access_token string
	Userid       string
	Urls         map[string]string
}

type StatusResponse struct {
	Structure map[string]interface{}
	Device    map[string]interface{}
	Shared    map[string]interface{}
	Metadata  map[string]interface{}
}

type DeviceDetails struct {
	Id             string
	Timestamp      float64
	TargetHumidity float64
	CurrHumidity   float64
	Name           string
	TargetTempType string
	TargetTemp     float64
	TargetTempLow  float64
	TargetTempHigh float64
	CurrTemp       float64
}

type StructureDetails struct {
	Id            string
	Name          string
	Timestamp     float64
	Away          bool
	Location      string
	PostalCode    string
	StreetAddress string
	Devices       []DeviceDetails
}

type ParsedStatus []StructureDetails

func (s ParsedStatus) String() string {
	ret := ""
	for i := range s {
		ret += fmt.Sprintln(s[i].Name)
		for j := range s[i].Devices {
			ret += fmt.Sprintf("\tDevice: %s\n", s[i].Devices[j].Name)
			ret += fmt.Sprintf("\t\tTime: %s\n", time.Unix(int64(s[i].Devices[j].Timestamp/1000.0), 0))
			ret += fmt.Sprintf("\t\tCurrTemp: %2.1f\n", s[i].Devices[j].CurrTemp)
			ret += fmt.Sprintf("\t\tCurrHumidity: %2.1f\n\n", s[i].Devices[j].CurrHumidity)
		}
	}
	return ret
}

func parseStatusResponse(s *StatusResponse) (ParsedStatus, error) {
	status := make([]StructureDetails, 0)
	structures := s.Structure
	devices := s.Device
	shared := s.Shared
	metadata := s.Metadata

	for struct_key, struct_vals := range structures {
		structDetails := StructureDetails{}
		structDetails.Id = struct_key

		valMap := struct_vals.(map[string]interface{})
		structDetails.Name = valMap["name"].(string)
		structDetails.Timestamp = valMap["$timestamp"].(float64)
		structDetails.Away = valMap["away"].(bool)
		structDetails.Location = valMap["location"].(string)
		structDetails.PostalCode = valMap["postal_code"].(string)
		structDetails.StreetAddress = valMap["street_address"].(string)

		structDevices := valMap["devices"].([]interface{})
		for device := range structDevices {
			devDetails := DeviceDetails{}
			devId := structDevices[device].(string)[7:]

			devVals := devices[devId].(map[string]interface{})

			devDetails.Id = devId
			//devDetails.Timestamp = devVals["$timestamp"].(float64)
			devDetails.CurrHumidity = devVals["current_humidity"].(float64)
			devDetails.TargetHumidity = devVals["target_humidity"].(float64)

			sharedVals := shared[devId].(map[string]interface{})
			metaVals := metadata[devId].(map[string]interface{})

			devDetails.Id = devId
			devDetails.Timestamp = metaVals["$timestamp"].(float64)
			devDetails.CurrTemp = sharedVals["current_temperature"].(float64)
			devDetails.TargetTemp = sharedVals["target_temperature"].(float64)
			devDetails.Name = sharedVals["name"].(string)
			devDetails.TargetTempType = sharedVals["target_temperature_type"].(string)
			devDetails.TargetTempHigh = sharedVals["target_temperature_high"].(float64)
			devDetails.TargetTempLow = sharedVals["target_temperature_low"].(float64)

			structDetails.Devices = append(structDetails.Devices, devDetails)

		}

		status = append(status, structDetails)
	}

	return status, nil
}

func NewNest(_username, _password string) *Nest {
	return &Nest{
		username:       _username,
		password:       _password,
		access_token:   "",
		userid:         "",
		transport_url:  "",
		client:         nil,
		status_request: nil,
	}
}

func (n *Nest) Login() (map[string]string, error) {

	data := url.Values{"username": {n.username}, "password": {n.password}}

	resp, err := http.PostForm("https://home.nest.com/user/login", data)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	parseErr := json.Unmarshal(body, &loginResp)

	n.transport_url = loginResp.Urls["transport_url"]
	n.access_token = loginResp.Access_token
	n.userid = loginResp.Userid

	n.client = &http.Client{}

	return map[string]string{
			"transport_url": loginResp.Urls["transport_url"],
			"access_token":  loginResp.Access_token,
			"userid":        loginResp.Userid,
		},
		parseErr
}

func (n *Nest) GetStatus() (ParsedStatus, error) {
	if n.client != nil {
		if n.status_request != nil {
			res, err := n.client.Do(n.status_request)

			defer res.Body.Close()

			if err != nil {
				return nil, err
			}

			body, err := ioutil.ReadAll(res.Body)

			var status_response StatusResponse
			json.Unmarshal(body, &status_response)

			status, _ := parseStatusResponse(&status_response)

			return status, err
		} else {
			status_url := n.transport_url + "/v2/mobile/user." + n.userid
			authorization := "Basic " + n.access_token

			req, _ := http.NewRequest("GET", status_url, nil)
			req.Header.Add("Authorization", authorization)
			req.Header.Add("X-nl-user-id", n.userid)
			req.Header.Add("X-nl-protocol-version", "1")
			n.status_request = req

			res, err := n.client.Do(req)
			defer res.Body.Close()

			if err != nil {
				return nil, err
			}

			body, err := ioutil.ReadAll(res.Body)

			var status_response StatusResponse
			json.Unmarshal(body, &status_response)

			status, _ := parseStatusResponse(&status_response)

			return status, err
		}
	}
	return nil, errors.New("No login credentials found, login before calling GetStatus")
}
