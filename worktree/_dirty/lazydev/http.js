

fetch("https://"+document.location.host+"/",{mode: 'no-cors'}).then(resp=>{
    document.location.protocol = "https:"
}).catch(err=>{
    console.log("The https certificate is not installed")
})
