/*
    Demo for showing agi script running in subservice

*/
//Request access to filelib
if (requirelib("filelib")){
    //Scan the user desktop
    var filelist = filelib.glob("user:/Desktop/*")
    sendJSONResp(JSON.stringify(filelist));
}else{
    sendJSONResp(JSON.stringify({
        error: "Filelib require failed"
    }));
}
