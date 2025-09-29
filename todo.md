# pentesting plan


we might need a state

```json
{
    "current_step": 1,
    "servers": ["172.17.0.1"],
    "ports": [
        "172.17.0.1:3000",
        "172.17.0.1:8080"
    ],
    "apps" : {
        "172.17.0.1:3000" : {
            "name": "juice shop",
            "version": "1.2.3",
            "vulns": [
                {"id":"GHSA-vf2r-cxg9-p7rf", "notes": "..."}
            ]
        },
        "172.17.0.1:8080": {
            "name": "wordpress",
            "version": "2.1.0",
            "vulns": [
                {"id":"RLSA-2025:7432", "notes": "..."}
            ]
        }
    },
}
```


# process
1. we pick server to check and put int .servers
2. we find open ports for the servers and put into .ports
3. find which apps are running on those ports and their versions and put into .apps
5. loop through vulnerabillities to mark all of them, into a vulnerabillities list. use curl and a osv.dev website to check.
6. check if this vulnerabillity applies and active