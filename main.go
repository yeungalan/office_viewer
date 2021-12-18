package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	aroz "imuslab.com/arozos/officeviewer/aroz"
)

var (
	handler  *aroz.ArozHandler
	domain   string
	publicIp string
)

type exchangeInfoStruct struct {
	Domain     string `json:"domain"`
	FileShared string `json:"fileShared"`
	UUID       string `json:"UUID"`
	IP         string `json:"ip"`
	Status     string `json:"status"`
	Error      string `json:"error"`
}

/*
	Demo for showing the implementation of ArOZ Online Subservice Structure

	Proxy url is get from filepath.Dir(StartDir) of the serviceInfo.
	In this example, the proxy path is demo/*
*/

//Kill signal handler. Do something before the system the core terminate.
func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("\r- Shutting down office viewer module.")
		//Do other things like close database or opened files

		os.Exit(0)
	}()
}

func main() {
	//If you have other flags, please add them here

	//Start the aoModule pipeline (which will parse the flags as well). Pass in the module launch information
	handler = aroz.HandleFlagParse(aroz.ServiceInfo{
		Name:     "Office Viewer",
		Desc:     "Simple office viewer",
		Group:    "Development",
		IconPath: "OfficeViewer/icon.png",
		Version:  "0.0.1",
		//You can define any path before the actualy html file. This directory (in this case demo/ ) will be the reverse proxy endpoint for this module
		StartDir:     "OfficeViewer/home.html",
		SupportFW:    true,
		LaunchFWDir:  "OfficeViewer/home.html",
		SupportEmb:   true,
		LaunchEmb:    "OfficeViewer/embedded.html",
		InitFWSize:   []int{720, 480},
		InitEmbSize:  []int{720, 480},
		SupportedExt: []string{".docx", ".pptx", ".xlsx"},
	})

	//get the domain
	publicIp = strings.TrimSpace(getPublicIP())
	domain = GetFreeDomain(publicIp)[0] //only the first domain is ok =D
	domain = strings.TrimSuffix(domain, ".")

	//Register the standard web services urls
	fs := http.FileServer(http.Dir("./web"))
	http.HandleFunc("/exchangeInfo", exchangeInfo)
	http.Handle("/", fs)

	//To receive kill signal from the System core, you can setup a close handler to catch the kill signal
	//This is not nessary if you have no opened files / database running
	SetupCloseHandler()

	//Any log println will be shown in the core system via STDOUT redirection. But not STDIN.
	log.Println("Office viewer module started. Listening on " + handler.Port)
	err := http.ListenAndServe(handler.Port, nil)
	if err != nil {
		log.Fatal(err)
	}

}

//API Test Demo. This showcase how can you access arozos resources with RESTFUL API CALL
func exchangeInfo(w http.ResponseWriter, r *http.Request) {
	//Get username and token from request
	_, token := handler.GetUserInfoFromRequest(w, r)
	//get the path
	path, err := mv(r, "path", false)

	//create the struct for return value
	returnJSONStruct := exchangeInfoStruct{
		Domain:     domain,
		FileShared: path,
		IP:         publicIp,
		UUID:       "",
		Status:     "unknown",
		Error:      "",
	}
	if err != nil {
		//Something went wrong when performing POST request
		returnJSONStruct.Status = "fail"
		returnJSONStruct.Error = err.Error()
		log.Println(err)
	}

	//Create an AGI Call that get the user desktop files
	script := `
		if (requirelib("share")){
			sendJSONResp(share.shareFile("` + path + `", 600)); //File virtual path and timeout in seconds.
		}else{
			sendJSONResp("Error");
		}
	`

	//Execute the AGI request on server side
	resp, err := handler.RequestGatewayInterface(token, script)
	if err != nil {
		//Something went wrong when performing POST request
		returnJSONStruct.Status = "fail"
		returnJSONStruct.Error = err.Error()
		log.Println(err)
	} else {
		//Try to read the resp body
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			returnJSONStruct.Status = "fail"
			returnJSONStruct.Error = err.Error()
			log.Println(err)
		}
		resp.Body.Close()
		//Check if it hv error
		if string(bodyBytes) == "Error" {
			returnJSONStruct.Status = "fail"
			returnJSONStruct.Error = "AGI Execution error!"
			log.Println(err)
		} else {
			//Check if the UUID is vaild, if not, return error
			if IsValidUUID(string(bodyBytes)) {
				returnJSONStruct.UUID = string(bodyBytes)
			} else {
				returnJSONStruct.Status = "fail"
				returnJSONStruct.Error = string(bodyBytes)
			}
		}

		//if the program didn't fail in any stage, aka it is success
		if returnJSONStruct.Status == "unknown" {
			returnJSONStruct.Status = "success"
			returnJSONStruct.Error = ""
		}
		//make it become json string
		jsonString, err := json.Marshal(returnJSONStruct)
		if err != nil {
			//Errr even it hv error it can't return to client XD
			log.Println(err)
		}
		sendJSONResponse(w, string(jsonString))
	}
}

//get the public IP from amazon, by TC
func getPublicIP() string {
	resp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//Convert the body to type string
	sb := string(body)
	return sb
}

//get the domain from DNS, by TC
func GetFreeDomain(ipaddr string) []string {
	ptr, _ := net.LookupAddr(ipaddr)
	return ptr
}

//check the vaild UUID
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
