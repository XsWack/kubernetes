curl POST https://iam.cn-north-1.myhuaweicloud.com/v3/auth/tokens -H 'content-type: application/json' -v -k -d '{
    "auth": {
        "identity": {
            "methods": [
                "password"
            ],
            "password": {
                "user": {
                    "name": "hwstaff_m00355558",
                    "domain": {
                        "name": "hwstaff_m00355558"
                    },
                    "password": "Huawei@123"
                }
            }
        },
        "scope": {
            "project": {
                "name": "cn-north-1"
            }
        }
    }
}'


curl GET https://cci.cn-north-1.myhuaweicloud.com/api/v1/namespaces/imagepullns/pods/test-88d8c64cc-d8q4b  -H 'content-type: application/json' -H "x-auth-token: $Token" -v -k


curl GET https://cci.cn-north-1.myhuaweicloud.com/apis/apps/v1/namespaces/imagepullns/deployments/test  -H 'content-type: application/json' -H "x-auth-token: $Token" -v -k