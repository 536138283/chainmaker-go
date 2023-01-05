/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/require"
)

var testIBCAuthType = "permissionedwithibc"

var testIBCOrg1 = "wx-org1.chainmaker.org"
var testIBCOrg2 = "wx-org2.chainmaker.org"
var testIBCOrg3 = "wx-org3.chainmaker.org"
var testIBCOrg4 = "wx-org4.chainmaker.org"
var testIBCOrg5 = "wx-org5.chainmaker.org"

var testIBCCertOrg1 = `-----BEGIN CERTIFICATE-----
MIICnjCCAkSgAwIBAgIDAml1MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMTEyOTAzMDUzOVoXDTMy
MTEyNjAzMDUzOVowgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARLK3ie2YGRntWtRMcW
Z9weScDAOsyhOim1/vGVv9ikONZd6nFwKfAMmW66T2owhXruK1UIAVZDsGmFl2lh
QVWto4GWMIGTMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MCkGA1Ud
DgQiBCBQ6DDQ/cYi1hUvNxliwIEjPE9pgnGMR7CkR4rlaw5/FzBFBgNVHREEPjA8
gg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmcxLmNoYWlubWFr
ZXIub3JnhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIBO8tvA1HYDlMjDJR+nur6Ej
YJcriJ0B4mtA3yH2rv8bAiEA9XFEM5rRl1F9G1828le9M4rb4iWp30Or1npaUYJg
66c=
-----END CERTIFICATE-----`

var testIBCConsensusTLSCert1 = `-----BEGIN CERTIFICATE-----
MIIC8jCCApigAwIBAgIDDhIAMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMTEyOTAzMDUzOVoXDTMy
MTEyNjAzMDUzOVowgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRMwEQYDVQQLEwpDaGFpbk1ha2VyMS4wLAYDVQQDEyVjb25zZW5zdXMxLnRscy53
eC1vcmcxLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
3vfgJFtbbrxDmZrfy9h0npTwLxQbAutS1mWIYZGIiKsqXZToOi2rk1G8AOTf54HR
J/ERDwochZmEFoahdQESHKOB3TCB2jAOBgNVHQ8BAf8EBAMCA/gwHQYDVR0lBBYw
FAYIKwYBBQUHAwEGCCsGAQUFBwMCMCkGA1UdDgQiBCCC1rRM9ufYbhxgOcJCERTH
ge7Aq33P4fdawrUyv0+XUjArBgNVHSMEJDAigCBQ6DDQ/cYi1hUvNxliwIEjPE9p
gnGMR7CkR4rlaw5/FzBRBgNVHREESjBIgg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxo
b3N0giVjb25zZW5zdXMxLnRscy53eC1vcmcxLmNoYWlubWFrZXIub3JnhwR/AAAB
MAoGCCqGSM49BAMCA0gAMEUCIQDHWgjiMGqdkpAWpnCt+qHUK1KR7Dya+FXvvt7P
8FdcwQIgNrqD1xaVIMnx0d6IYP0nFjUfnbCGJYRRzXvnr7yrNZE=
-----END CERTIFICATE-----`

var testIBCMasterPKOrg1 = `-----BEGIN MASTER PUBLIC KEY-----
MIGVMAoGCCqBHM9VAYIuA4GGAAOBggAERXAcV+mh3oqrpUNbiE694fUvJjBfJ+8j
GrlMDf3c0xcVrkTFiWQJsGyL6LfTD1wcs0aFOesB8HOBqiO8VE7abZcZRnDYE6Zi
l0PyhcbOR8oTRV7RM70nZzaDGP1GzwzcNgJ1V/UP5Le1srf1HyKazQXVn5U6ih4A
DFVUuex8iwE=
-----END MASTER PUBLIC KEY-----`

var testIBCMasterPKOrg2 = `-----BEGIN MASTER PUBLIC KEY-----
MIGVMAoGCCqBHM9VAYIuA4GGAAOBggAEDEeay/jSAf1wIhzubri/urcKmRPPIRE4
T+PhrXN1Tg+sr/W+cZaUewDNqnU3+P0VgUqLWU8C4jkR+tLBXFBr0Df92HvlCDu+
pQDfTYIvfENFyYAKvBgFdehwQ52OA6Z9KpFeviUgXi6RC8WY/Ebh0HJNTq7yFdzM
sYoHOq/+KYM=
-----END MASTER PUBLIC KEY-----`

var testIBCMasterPKOrg3 = `-----BEGIN MASTER PUBLIC KEY-----
MIGVMAoGCCqBHM9VAYIuA4GGAAOBggAEeTqP0IhOSmJyPgYL2gR/SJj7IRveQcke
dHeNbzUM+exSGWr5TFwFKfl8gieJQxp7A82u1q4sWgsC91L0sdz6IyEOierGLpTZ
UMB4Pabmr6gzNIhAxgpaNbi4lZpNJetJLDBdXU3duCEATLJdoqi5O61+MEDJIGFp
K+vZp5Epg4o=
-----END MASTER PUBLIC KEY-----`

var testIBCMasterPKOrg4 = `-----BEGIN MASTER PUBLIC KEY-----
MIGVMAoGCCqBHM9VAYIuA4GGAAOBggAEfgsHaVRNOWuoQOSaosYyiqGt0btxDiYu
v84tEllY0SopbluLTKiQ1OuHT93vYy4mPBFzqAAGOUOrS2FaIiHyUqYNfBxzXxEe
++QmJHB0/qrWGZhxNppv2PqCUycu4NOCMNsi9SLyYWWzWTge5D+JjYMwTbxkA4Jm
O9W+zAO18Jw=
-----END MASTER PUBLIC KEY-----`

var testIBCConsensusPK1 = `-----BEGIN PUBLIC KEY-----
MIHyMAoGCCqBHM9VAYIuA4HjADCB3wNUAGlkPWNvbnNlbnN1czE7b3JnPXd4LW9y
ZzEuY2hhaW5tYWtlci5vcmc7dHlwZT1jb25zZW5zdXM7dmI9MTAwMDAwMDAwMDt2
ZT05MDAwMDAwMDAwA4GGAAOBggAERXAcV+mh3oqrpUNbiE694fUvJjBfJ+8jGrlM
Df3c0xcVrkTFiWQJsGyL6LfTD1wcs0aFOesB8HOBqiO8VE7abZcZRnDYE6Zil0Py
hcbOR8oTRV7RM70nZzaDGP1GzwzcNgJ1V/UP5Le1srf1HyKazQXVn5U6ih4ADFVU
uex8iwE=
-----END PUBLIC KEY-----`

var testIBCConsensusPK2 = `-----BEGIN PUBLIC KEY-----
MIHyMAoGCCqBHM9VAYIuA4HjADCB3wNUAGlkPWNvbnNlbnN1czE7b3JnPXd4LW9y
ZzIuY2hhaW5tYWtlci5vcmc7dHlwZT1jb25zZW5zdXM7dmI9MTAwMDAwMDAwMDt2
ZT05MDAwMDAwMDAwA4GGAAOBggAEDEeay/jSAf1wIhzubri/urcKmRPPIRE4T+Ph
rXN1Tg+sr/W+cZaUewDNqnU3+P0VgUqLWU8C4jkR+tLBXFBr0Df92HvlCDu+pQDf
TYIvfENFyYAKvBgFdehwQ52OA6Z9KpFeviUgXi6RC8WY/Ebh0HJNTq7yFdzMsYoH
Oq/+KYM=
-----END PUBLIC KEY-----`

var testIBCConsensusPK3 = `-----BEGIN PUBLIC KEY-----
MIHyMAoGCCqBHM9VAYIuA4HjADCB3wNUAGlkPWNvbnNlbnN1czE7b3JnPXd4LW9y
ZzMuY2hhaW5tYWtlci5vcmc7dHlwZT1jb25zZW5zdXM7dmI9MTAwMDAwMDAwMDt2
ZT05MDAwMDAwMDAwA4GGAAOBggAEeTqP0IhOSmJyPgYL2gR/SJj7IRveQckedHeN
bzUM+exSGWr5TFwFKfl8gieJQxp7A82u1q4sWgsC91L0sdz6IyEOierGLpTZUMB4
Pabmr6gzNIhAxgpaNbi4lZpNJetJLDBdXU3duCEATLJdoqi5O61+MEDJIGFpK+vZ
p5Epg4o=
-----END PUBLIC KEY-----`

var testIBCConsensusPK4 = `-----BEGIN PUBLIC KEY-----
MIHyMAoGCCqBHM9VAYIuA4HjADCB3wNUAGlkPWNvbnNlbnN1czE7b3JnPXd4LW9y
ZzQuY2hhaW5tYWtlci5vcmc7dHlwZT1jb25zZW5zdXM7dmI9MTAwMDAwMDAwMDt2
ZT05MDAwMDAwMDAwA4GGAAOBggAEfgsHaVRNOWuoQOSaosYyiqGt0btxDiYuv84t
EllY0SopbluLTKiQ1OuHT93vYy4mPBFzqAAGOUOrS2FaIiHyUqYNfBxzXxEe++Qm
JHB0/qrWGZhxNppv2PqCUycu4NOCMNsi9SLyYWWzWTge5D+JjYMwTbxkA4JmO9W+
zAO18Jw=
-----END PUBLIC KEY-----`

var testIBCConsensusPK5 = `-----BEGIN PUBLIC KEY-----
MIHyMAoGCCqBHM9VAYIuA4HjADCB3wNUAGlkPWNvbnNlbnN1czE7b3JnPXd4LW9y
ZzUuY2hhaW5tYWtlci5vcmc7dHlwZT1jb25zZW5zdXM7dmI9MTAwMDAwMDAwMDt2
ZT05MDAwMDAwMDAwA4GGAAOBggAEdDwzHzf8tl9QwfTSuuCEMzDvuCMKwRQqWTtN
RwkW4dQ85SvvBP4EtjDI9ydCDDI2QEplixhwzOcklhcZDSFLRSUb3dcCSsDPr5MU
Bt2wx0EEjxMOKiJMhz6c02CdiDC1R5DTeMwJRdRhhzUevPYlERuEldNO9llVFKKq
v9Dz21U=
-----END PUBLIC KEY-----`

var testIBCConsensusSK1 = `-----BEGIN PRIVATE KEY-----
MIIBPQIBADAKBggqgRzPVQGCLgSCASowggEmA1QAaWQ9Y29uc2Vuc3VzMTtvcmc9
d3gtb3JnMS5jaGFpbm1ha2VyLm9yZzt0eXBlPWNvbnNlbnN1czt2Yj0xMDAwMDAw
MDAwO3ZlPTkwMDAwMDAwMDADRQADQgAESqyGA45F0ZMkOCzKWboMsKuRTVn9rhzu
rN67zECobmdO8TohO0leL/Og8tjGDrp4Bv8nnUPs5YUYs2afu6HDwAOBhgADgYIA
BEVwHFfpod6Kq6VDW4hOveH1LyYwXyfvIxq5TA393NMXFa5ExYlkCbBsi+i30w9c
HLNGhTnrAfBzgaojvFRO2m2XGUZw2BOmYpdD8oXGzkfKE0Ve0TO9J2c2gxj9Rs8M
3DYCdVf1D+S3tbK39R8ims0F1Z+VOooeAAxVVLnsfIsB
-----END PRIVATE KEY-----`

var testIBCConsensusSK2 = `-----BEGIN PRIVATE KEY-----
MIIBPQIBADAKBggqgRzPVQGCLgSCASowggEmA1QAaWQ9Y29uc2Vuc3VzMTtvcmc9
d3gtb3JnMi5jaGFpbm1ha2VyLm9yZzt0eXBlPWNvbnNlbnN1czt2Yj0xMDAwMDAw
MDAwO3ZlPTkwMDAwMDAwMDADRQADQgAELgV7Y8SJdODSW48fWUx/WUtlwUzWTCYw
iw3XpnPU3K6Y4AwRpRf0GdOlHrbBleMz7Pkr3vG2gtxEJZwOFqO63gOBhgADgYIA
BAxHmsv40gH9cCIc7m64v7q3CpkTzyEROE/j4a1zdU4PrK/1vnGWlHsAzap1N/j9
FYFKi1lPAuI5EfrSwVxQa9A3/dh75Qg7vqUA302CL3xDRcmACrwYBXXocEOdjgOm
fSqRXr4lIF4ukQvFmPxG4dByTU6u8hXczLGKBzqv/imD
-----END PRIVATE KEY-----`

var testIBCConsensusSK3 = `-----BEGIN PRIVATE KEY-----
MIIBPQIBADAKBggqgRzPVQGCLgSCASowggEmA1QAaWQ9Y29uc2Vuc3VzMTtvcmc9
d3gtb3JnMy5jaGFpbm1ha2VyLm9yZzt0eXBlPWNvbnNlbnN1czt2Yj0xMDAwMDAw
MDAwO3ZlPTkwMDAwMDAwMDADRQADQgAEpb7QKs7O5QY60EkJ48uintUQXFLZxnLq
oYuMyvAOGXUx+G51fm2uiwCA4Gmnx9VrFyVY1Vs7TfYRBVZc12htGgOBhgADgYIA
BHk6j9CITkpicj4GC9oEf0iY+yEb3kHJHnR3jW81DPnsUhlq+UxcBSn5fIIniUMa
ewPNrtauLFoLAvdS9LHc+iMhDonqxi6U2VDAeD2m5q+oMzSIQMYKWjW4uJWaTSXr
SSwwXV1N3bghAEyyXaKouTutfjBAySBhaSvr2aeRKYOK
-----END PRIVATE KEY-----`

var testIBCConsensusSK4 = `-----BEGIN PRIVATE KEY-----
MIIBPQIBADAKBggqgRzPVQGCLgSCASowggEmA1QAaWQ9Y29uc2Vuc3VzMTtvcmc9
d3gtb3JnNC5jaGFpbm1ha2VyLm9yZzt0eXBlPWNvbnNlbnN1czt2Yj0xMDAwMDAw
MDAwO3ZlPTkwMDAwMDAwMDADRQADQgAEcowi6nDwBk5Ea2mhAqK8zVvMZDCvtqsF
JWB7oNXDVqGqizydSM8Rc4q0iJiBnhh0uvdTt8zSPUnL9OynfNgY4AOBhgADgYIA
BH4LB2lUTTlrqEDkmqLGMoqhrdG7cQ4mLr/OLRJZWNEqKW5bi0yokNTrh0/d72Mu
JjwRc6gABjlDq0thWiIh8lKmDXwcc18RHvvkJiRwdP6q1hmYcTaab9j6glMnLuDT
gjDbIvUi8mFls1k4HuQ/iY2DME28ZAOCZjvVvswDtfCc
-----END PRIVATE KEY-----`

var testIBCConsensusSK5 = `-----BEGIN PRIVATE KEY-----
MIIBPQIBADAKBggqgRzPVQGCLgSCASowggEmA1QAaWQ9Y29uc2Vuc3VzMTtvcmc9
d3gtb3JnNS5jaGFpbm1ha2VyLm9yZzt0eXBlPWNvbnNlbnN1czt2Yj0xMDAwMDAw
MDAwO3ZlPTkwMDAwMDAwMDADRQADQgAEFEhz4U4VE/HHAX9jRbLcV6yZLWaf67V0
gX0oDCb3yRuo33PeFiU0a39J8FRHgZMJOkhHQpsUTdXaj6mvOi+dIwOBhgADgYIA
BHQ8Mx83/LZfUMH00rrghDMw77gjCsEUKlk7TUcJFuHUPOUr7wT+BLYwyPcnQgwy
NkBKZYsYcMznJJYXGQ0hS0UlG93XAkrAz6+TFAbdsMdBBI8TDioiTIc+nNNgnYgw
tUeQ03jMCUXUYYc1Hrz2JREbhJXTTvZZVRSiqr/Q89tV
-----END PRIVATE KEY-----`

var testIBCAdminPK1 = `-----BEGIN PUBLIC KEY-----
MIHqMAoGCCqBHM9VAYIuA4HbADCB1wNMAGlkPWFkbWluMTtvcmc9d3gtb3JnMS5j
aGFpbm1ha2VyLm9yZzt0eXBlPWFkbWluO3ZiPTEwMDAwMDAwMDA7dmU9OTAwMDAw
MDAwMAOBhgADgYIABEVwHFfpod6Kq6VDW4hOveH1LyYwXyfvIxq5TA393NMXFa5E
xYlkCbBsi+i30w9cHLNGhTnrAfBzgaojvFRO2m2XGUZw2BOmYpdD8oXGzkfKE0Ve
0TO9J2c2gxj9Rs8M3DYCdVf1D+S3tbK39R8ims0F1Z+VOooeAAxVVLnsfIsB
-----END PUBLIC KEY-----`

var testIBCAdminPK2 = `-----BEGIN PUBLIC KEY-----
MIHqMAoGCCqBHM9VAYIuA4HbADCB1wNMAGlkPWFkbWluMTtvcmc9d3gtb3JnMi5j
aGFpbm1ha2VyLm9yZzt0eXBlPWFkbWluO3ZiPTEwMDAwMDAwMDA7dmU9OTAwMDAw
MDAwMAOBhgADgYIABAxHmsv40gH9cCIc7m64v7q3CpkTzyEROE/j4a1zdU4PrK/1
vnGWlHsAzap1N/j9FYFKi1lPAuI5EfrSwVxQa9A3/dh75Qg7vqUA302CL3xDRcmA
CrwYBXXocEOdjgOmfSqRXr4lIF4ukQvFmPxG4dByTU6u8hXczLGKBzqv/imD
-----END PUBLIC KEY-----`

var testIBCAdminPK3 = `-----BEGIN PUBLIC KEY-----
MIHqMAoGCCqBHM9VAYIuA4HbADCB1wNMAGlkPWFkbWluMTtvcmc9d3gtb3JnMy5j
aGFpbm1ha2VyLm9yZzt0eXBlPWFkbWluO3ZiPTEwMDAwMDAwMDA7dmU9OTAwMDAw
MDAwMAOBhgADgYIABHk6j9CITkpicj4GC9oEf0iY+yEb3kHJHnR3jW81DPnsUhlq
+UxcBSn5fIIniUMaewPNrtauLFoLAvdS9LHc+iMhDonqxi6U2VDAeD2m5q+oMzSI
QMYKWjW4uJWaTSXrSSwwXV1N3bghAEyyXaKouTutfjBAySBhaSvr2aeRKYOK
-----END PUBLIC KEY-----`

var testIBCAdminPK4 = `-----BEGIN PUBLIC KEY-----
MIHqMAoGCCqBHM9VAYIuA4HbADCB1wNMAGlkPWFkbWluMTtvcmc9d3gtb3JnNC5j
aGFpbm1ha2VyLm9yZzt0eXBlPWFkbWluO3ZiPTEwMDAwMDAwMDA7dmU9OTAwMDAw
MDAwMAOBhgADgYIABH4LB2lUTTlrqEDkmqLGMoqhrdG7cQ4mLr/OLRJZWNEqKW5b
i0yokNTrh0/d72MuJjwRc6gABjlDq0thWiIh8lKmDXwcc18RHvvkJiRwdP6q1hmY
cTaab9j6glMnLuDTgjDbIvUi8mFls1k4HuQ/iY2DME28ZAOCZjvVvswDtfCc
-----END PUBLIC KEY-----`

var testIBCAdminPK5 = `-----BEGIN PUBLIC KEY-----
MIHqMAoGCCqBHM9VAYIuA4HbADCB1wNMAGlkPWFkbWluMTtvcmc9d3gtb3JnNS5j
aGFpbm1ha2VyLm9yZzt0eXBlPWFkbWluO3ZiPTEwMDAwMDAwMDA7dmU9OTAwMDAw
MDAwMAOBhgADgYIABHQ8Mx83/LZfUMH00rrghDMw77gjCsEUKlk7TUcJFuHUPOUr
7wT+BLYwyPcnQgwyNkBKZYsYcMznJJYXGQ0hS0UlG93XAkrAz6+TFAbdsMdBBI8T
DioiTIc+nNNgnYgwtUeQ03jMCUXUYYc1Hrz2JREbhJXTTvZZVRSiqr/Q89tV
-----END PUBLIC KEY-----`

var testIBCAdminSK1 = `-----BEGIN PRIVATE KEY-----
MIIBNQIBADAKBggqgRzPVQGCLgSCASIwggEeA0wAaWQ9YWRtaW4xO29yZz13eC1v
cmcxLmNoYWlubWFrZXIub3JnO3R5cGU9YWRtaW47dmI9MTAwMDAwMDAwMDt2ZT05
MDAwMDAwMDAwA0UAA0IABJ4//nZPiBapDryAkPh7JxX04arMqHLQ9MY7utNxQUr3
WxoDHIwFf8Yde4joFkq6d8jS+NQgZyAHABCA3aH2/7IDgYYAA4GCAARFcBxX6aHe
iqulQ1uITr3h9S8mMF8n7yMauUwN/dzTFxWuRMWJZAmwbIvot9MPXByzRoU56wHw
c4GqI7xUTtptlxlGcNgTpmKXQ/KFxs5HyhNFXtEzvSdnNoMY/UbPDNw2AnVX9Q/k
t7Wyt/UfIprNBdWflTqKHgAMVVS57HyLAQ==
-----END PRIVATE KEY-----`

var testIBCAdminSK2 = `-----BEGIN PRIVATE KEY-----
MIIBNQIBADAKBggqgRzPVQGCLgSCASIwggEeA0wAaWQ9YWRtaW4xO29yZz13eC1v
cmcyLmNoYWlubWFrZXIub3JnO3R5cGU9YWRtaW47dmI9MTAwMDAwMDAwMDt2ZT05
MDAwMDAwMDAwA0UAA0IABJPi6/c1ljS+X1ldcLwXuyynkS3xDnNByL0ULgXphpBj
mN9QYytTl439q0litAGuNFJ8FBKpj9NkEixZuS0k9AoDgYYAA4GCAAQMR5rL+NIB
/XAiHO5uuL+6twqZE88hEThP4+Gtc3VOD6yv9b5xlpR7AM2qdTf4/RWBSotZTwLi
ORH60sFcUGvQN/3Ye+UIO76lAN9Ngi98Q0XJgAq8GAV16HBDnY4Dpn0qkV6+JSBe
LpELxZj8RuHQck1OrvIV3Myxigc6r/4pgw==
-----END PRIVATE KEY-----`

var testIBCAdminSK3 = `-----BEGIN PRIVATE KEY-----
MIIBNQIBADAKBggqgRzPVQGCLgSCASIwggEeA0wAaWQ9YWRtaW4xO29yZz13eC1v
cmczLmNoYWlubWFrZXIub3JnO3R5cGU9YWRtaW47dmI9MTAwMDAwMDAwMDt2ZT05
MDAwMDAwMDAwA0UAA0IABIPWdDxgOZbtaVjB/8eMYO7DwvPEGyiOu1HYp4mW3jXj
DEJQup2kxIjJRQ0FyxTmNDtfmN29XTgl2RYYFsV5SwUDgYYAA4GCAAR5Oo/QiE5K
YnI+BgvaBH9ImPshG95ByR50d41vNQz57FIZavlMXAUp+XyCJ4lDGnsDza7Wrixa
CwL3UvSx3PojIQ6J6sYulNlQwHg9puavqDM0iEDGClo1uLiVmk0l60ksMF1dTd24
IQBMsl2iqLk7rX4wQMkgYWkr69mnkSmDig==
-----END PRIVATE KEY-----`

var testIBCAdminSK4 = `-----BEGIN PRIVATE KEY-----
MIIBNQIBADAKBggqgRzPVQGCLgSCASIwggEeA0wAaWQ9YWRtaW4xO29yZz13eC1v
cmc0LmNoYWlubWFrZXIub3JnO3R5cGU9YWRtaW47dmI9MTAwMDAwMDAwMDt2ZT05
MDAwMDAwMDAwA0UAA0IABIqe2hZ4g32vPwUPzx5Bkbu1Sk5KykuK+Nf2utVfmBz6
bie0QNIPRo+w/f00f8RHU10gM8NyT9O2sS5X7Vi0OHwDgYYAA4GCAAR+CwdpVE05
a6hA5JqixjKKoa3Ru3EOJi6/zi0SWVjRKiluW4tMqJDU64dP3e9jLiY8EXOoAAY5
Q6tLYVoiIfJSpg18HHNfER775CYkcHT+qtYZmHE2mm/Y+oJTJy7g04Iw2yL1IvJh
ZbNZOB7kP4mNgzBNvGQDgmY71b7MA7XwnA==
-----END PRIVATE KEY-----`
var testIBCAdminSK5 = `-----BEGIN PRIVATE KEY-----
MIIBNQIBADAKBggqgRzPVQGCLgSCASIwggEeA0wAaWQ9YWRtaW4xO29yZz13eC1v
cmc1LmNoYWlubWFrZXIub3JnO3R5cGU9YWRtaW47dmI9MTAwMDAwMDAwMDt2ZT05
MDAwMDAwMDAwA0UAA0IABLLe7a9XmUX76NsAWbiG4ma1nGCPIIoHwLqHJaJshmJ3
B8SwA6xq+bG8m6nCqhf3qNUu6XHTXrkMjboJjx0kXRkDgYYAA4GCAAR0PDMfN/y2
X1DB9NK64IQzMO+4IwrBFCpZO01HCRbh1DzlK+8E/gS2MMj3J0IMMjZASmWLGHDM
5ySWFxkNIUtFJRvd1wJKwM+vkxQG3bDHQQSPEw4qIkyHPpzTYJ2IMLVHkNN4zAlF
1GGHNR689iURG4SV0072WVUUoqq/0PPbVQ==
-----END PRIVATE KEY-----`

var testIBCClientPK1 = `-----BEGIN PUBLIC KEY-----
MIHsMAoGCCqBHM9VAYIuA4HdADCB2QNOAGlkPWNsaWVudDE7b3JnPXd4LW9yZzEu
Y2hhaW5tYWtlci5vcmc7dHlwZT1jbGllbnQ7dmI9MTAwMDAwMDAwMDt2ZT05MDAw
MDAwMDAwA4GGAAOBggAERXAcV+mh3oqrpUNbiE694fUvJjBfJ+8jGrlMDf3c0xcV
rkTFiWQJsGyL6LfTD1wcs0aFOesB8HOBqiO8VE7abZcZRnDYE6Zil0PyhcbOR8oT
RV7RM70nZzaDGP1GzwzcNgJ1V/UP5Le1srf1HyKazQXVn5U6ih4ADFVUuex8iwE=
-----END PUBLIC KEY-----`

var testIBCClientPK2 = `-----BEGIN PUBLIC KEY-----
MIHsMAoGCCqBHM9VAYIuA4HdADCB2QNOAGlkPWNsaWVudDE7b3JnPXd4LW9yZzIu
Y2hhaW5tYWtlci5vcmc7dHlwZT1jbGllbnQ7dmI9MTAwMDAwMDAwMDt2ZT05MDAw
MDAwMDAwA4GGAAOBggAEDEeay/jSAf1wIhzubri/urcKmRPPIRE4T+PhrXN1Tg+s
r/W+cZaUewDNqnU3+P0VgUqLWU8C4jkR+tLBXFBr0Df92HvlCDu+pQDfTYIvfENF
yYAKvBgFdehwQ52OA6Z9KpFeviUgXi6RC8WY/Ebh0HJNTq7yFdzMsYoHOq/+KYM=
-----END PUBLIC KEY-----`

var testIBCClientPK3 = `-----BEGIN PUBLIC KEY-----
MIHsMAoGCCqBHM9VAYIuA4HdADCB2QNOAGlkPWNsaWVudDE7b3JnPXd4LW9yZzMu
Y2hhaW5tYWtlci5vcmc7dHlwZT1jbGllbnQ7dmI9MTAwMDAwMDAwMDt2ZT05MDAw
MDAwMDAwA4GGAAOBggAEeTqP0IhOSmJyPgYL2gR/SJj7IRveQckedHeNbzUM+exS
GWr5TFwFKfl8gieJQxp7A82u1q4sWgsC91L0sdz6IyEOierGLpTZUMB4Pabmr6gz
NIhAxgpaNbi4lZpNJetJLDBdXU3duCEATLJdoqi5O61+MEDJIGFpK+vZp5Epg4o=
-----END PUBLIC KEY-----`

var testIBCClientPK4 = `-----BEGIN PUBLIC KEY-----
MIHsMAoGCCqBHM9VAYIuA4HdADCB2QNOAGlkPWNsaWVudDE7b3JnPXd4LW9yZzQu
Y2hhaW5tYWtlci5vcmc7dHlwZT1jbGllbnQ7dmI9MTAwMDAwMDAwMDt2ZT05MDAw
MDAwMDAwA4GGAAOBggAEfgsHaVRNOWuoQOSaosYyiqGt0btxDiYuv84tEllY0Sop
bluLTKiQ1OuHT93vYy4mPBFzqAAGOUOrS2FaIiHyUqYNfBxzXxEe++QmJHB0/qrW
GZhxNppv2PqCUycu4NOCMNsi9SLyYWWzWTge5D+JjYMwTbxkA4JmO9W+zAO18Jw=
-----END PUBLIC KEY-----`
var testIBCClientPK5 = `-----BEGIN PUBLIC KEY-----
MIHsMAoGCCqBHM9VAYIuA4HdADCB2QNOAGlkPWNsaWVudDE7b3JnPXd4LW9yZzUu
Y2hhaW5tYWtlci5vcmc7dHlwZT1jbGllbnQ7dmI9MTAwMDAwMDAwMDt2ZT05MDAw
MDAwMDAwA4GGAAOBggAEdDwzHzf8tl9QwfTSuuCEMzDvuCMKwRQqWTtNRwkW4dQ8
5SvvBP4EtjDI9ydCDDI2QEplixhwzOcklhcZDSFLRSUb3dcCSsDPr5MUBt2wx0EE
jxMOKiJMhz6c02CdiDC1R5DTeMwJRdRhhzUevPYlERuEldNO9llVFKKqv9Dz21U=
-----END PUBLIC KEY-----`

var testIBCClientSK1 = `-----BEGIN PRIVATE KEY-----
MIIBNwIBADAKBggqgRzPVQGCLgSCASQwggEgA04AaWQ9Y2xpZW50MTtvcmc9d3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzt0eXBlPWNsaWVudDt2Yj0xMDAwMDAwMDAwO3Zl
PTkwMDAwMDAwMDADRQADQgAEZon8vQkbI3H/dqOLI4cq9te9uXEjt6kMYY2wnEKU
4dUgRNxTuYqzy3ulkAhPDKyCfgyTEsBqXE6961Hf6p8vegOBhgADgYIABEVwHFfp
od6Kq6VDW4hOveH1LyYwXyfvIxq5TA393NMXFa5ExYlkCbBsi+i30w9cHLNGhTnr
AfBzgaojvFRO2m2XGUZw2BOmYpdD8oXGzkfKE0Ve0TO9J2c2gxj9Rs8M3DYCdVf1
D+S3tbK39R8ims0F1Z+VOooeAAxVVLnsfIsB
-----END PRIVATE KEY-----`

var testIBCClientSK2 = `-----BEGIN PRIVATE KEY-----
MIIBNwIBADAKBggqgRzPVQGCLgSCASQwggEgA04AaWQ9Y2xpZW50MTtvcmc9d3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzt0eXBlPWNsaWVudDt2Yj0xMDAwMDAwMDAwO3Zl
PTkwMDAwMDAwMDADRQADQgAENWiiyEnCvLL1hRv52rjRp5pROZ0h2Zn2Bf47fPYO
WD0AoK1Kn5sLeAcltilAY9UGVrf6tvCZrz3Cv/ULSt+ZegOBhgADgYIABAxHmsv4
0gH9cCIc7m64v7q3CpkTzyEROE/j4a1zdU4PrK/1vnGWlHsAzap1N/j9FYFKi1lP
AuI5EfrSwVxQa9A3/dh75Qg7vqUA302CL3xDRcmACrwYBXXocEOdjgOmfSqRXr4l
IF4ukQvFmPxG4dByTU6u8hXczLGKBzqv/imD
-----END PRIVATE KEY-----`

var testIBCClientSK3 = `-----BEGIN PRIVATE KEY-----
MIIBNwIBADAKBggqgRzPVQGCLgSCASQwggEgA04AaWQ9Y2xpZW50MTtvcmc9d3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzt0eXBlPWNsaWVudDt2Yj0xMDAwMDAwMDAwO3Zl
PTkwMDAwMDAwMDADRQADQgAEWCA2i5glOvlr5V4Y32Ia+ZrdDwAXNl05XdmN6aFN
JYor8tAJwuL2yOlk464WsGohEX9CCtOT5Lv1iWJG0qFy0wOBhgADgYIABHk6j9CI
Tkpicj4GC9oEf0iY+yEb3kHJHnR3jW81DPnsUhlq+UxcBSn5fIIniUMaewPNrtau
LFoLAvdS9LHc+iMhDonqxi6U2VDAeD2m5q+oMzSIQMYKWjW4uJWaTSXrSSwwXV1N
3bghAEyyXaKouTutfjBAySBhaSvr2aeRKYOK
-----END PRIVATE KEY-----`

var testIBCClientSK4 = `-----BEGIN PRIVATE KEY-----
MIIBNwIBADAKBggqgRzPVQGCLgSCASQwggEgA04AaWQ9Y2xpZW50MTtvcmc9d3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzt0eXBlPWNsaWVudDt2Yj0xMDAwMDAwMDAwO3Zl
PTkwMDAwMDAwMDADRQADQgAEpaJ/1WBsXNOdXg0oVNp2AO7rRE8DxP8mmVmFJkN/
ZSmGS3/xHRde4tMVlkjNZ9pxGsAE3qv5BzQn6pgcL1j+GwOBhgADgYIABH4LB2lU
TTlrqEDkmqLGMoqhrdG7cQ4mLr/OLRJZWNEqKW5bi0yokNTrh0/d72MuJjwRc6gA
BjlDq0thWiIh8lKmDXwcc18RHvvkJiRwdP6q1hmYcTaab9j6glMnLuDTgjDbIvUi
8mFls1k4HuQ/iY2DME28ZAOCZjvVvswDtfCc
-----END PRIVATE KEY-----`
var testIBCClientSK5 = `-----BEGIN PRIVATE KEY-----
MIIBNwIBADAKBggqgRzPVQGCLgSCASQwggEgA04AaWQ9Y2xpZW50MTtvcmc9d3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzt0eXBlPWNsaWVudDt2Yj0xMDAwMDAwMDAwO3Zl
PTkwMDAwMDAwMDADRQADQgAEHZ+sU/0mKjNKXLyzktRfC0abw4tELJufJtvtysGv
sxAOQJUbxEH44QaHT48U7WvkZcgrvpoVkxx/rDvvdViE/AOBhgADgYIABHQ8Mx83
/LZfUMH00rrghDMw77gjCsEUKlk7TUcJFuHUPOUr7wT+BLYwyPcnQgwyNkBKZYsY
cMznJJYXGQ0hS0UlG93XAkrAz6+TFAbdsMdBBI8TDioiTIc+nNNgnYgwtUeQ03jM
CUXUYYc1Hrz2JREbhJXTTvZZVRSiqr/Q89tV
-----END PRIVATE KEY-----`

var testIBCChainConfig = &config.ChainConfig{
	ChainId:  testChainId,
	Version:  testVersion,
	AuthType: testIBCAuthType,
	Sequence: 0,
	Crypto:   &config.CryptoConfig{Hash: testHashType},
	Block:    nil,
	Core:     nil,
	Consensus: &config.ConsensusConfig{
		Nodes: []*config.OrgConfig{
			{OrgId: testIBCOrg1},
			{OrgId: testIBCOrg2},
			{OrgId: testIBCOrg3},
			{OrgId: testIBCOrg4},
		},
	},
	TrustRoots: []*config.TrustRootConfig{
		{OrgId: testIBCOrg1, Root: []string{testIBCCertOrg1}},
		{OrgId: testIBCOrg2, Root: []string{testCAOrg2}},
		{OrgId: testIBCOrg3, Root: []string{testCAOrg3}},
		{OrgId: testIBCOrg4, Root: []string{testCAOrg4}},
	},
	TrustMembers: []*config.TrustMemberConfig{
		{OrgId: testOrg5, Role: "admin", MemberInfo: testTrustMember1},
		{OrgId: testOrg5, Role: "admin", MemberInfo: testTrustMember2},
	},
	IbcMasterKeys: []*config.IBCMasterKeyConfig{
		{OrgId: testIBCOrg1, MasterKeys: []string{testIBCMasterPKOrg1}},
		{OrgId: testIBCOrg2, MasterKeys: []string{testIBCMasterPKOrg2}},
		{OrgId: testIBCOrg3, MasterKeys: []string{testIBCMasterPKOrg3}},
		{OrgId: testIBCOrg4, MasterKeys: []string{testIBCMasterPKOrg4}},
	},
}

var ibcOrgMemberInfoMap = map[string]*orgMemberInfo{
	testIBCOrg1: {
		orgId:     testIBCOrg1,
		consensus: &testCertInfo{cert: testIBCConsensusPK1, sk: testIBCConsensusSK1},
		admin:     &testCertInfo{cert: testIBCAdminPK1, sk: testIBCAdminSK1},
		client:    &testCertInfo{cert: testIBCClientPK1, sk: testIBCClientSK1},
	},
	testIBCOrg2: {
		orgId:     testIBCOrg2,
		consensus: &testCertInfo{cert: testIBCConsensusPK2, sk: testIBCConsensusSK2},
		admin:     &testCertInfo{cert: testIBCAdminPK2, sk: testIBCAdminSK2},
		client:    &testCertInfo{cert: testIBCClientPK2, sk: testIBCClientSK2},
	},
	testIBCOrg3: {
		orgId:     testIBCOrg3,
		consensus: &testCertInfo{cert: testIBCConsensusPK3, sk: testIBCConsensusSK3},
		admin:     &testCertInfo{cert: testIBCAdminPK3, sk: testIBCAdminSK3},
		client:    &testCertInfo{cert: testIBCClientPK3, sk: testIBCClientSK3},
	},
	testIBCOrg4: {
		orgId:     testIBCOrg4,
		consensus: &testCertInfo{cert: testIBCConsensusPK4, sk: testIBCConsensusSK4},
		admin:     &testCertInfo{cert: testIBCAdminPK4, sk: testIBCAdminSK4},
		client:    &testCertInfo{cert: testIBCClientPK4, sk: testIBCClientSK4},
	},
	testIBCOrg5: {
		orgId:     testIBCOrg5,
		consensus: &testCertInfo{cert: testIBCConsensusPK5, sk: testIBCConsensusSK5},
		admin:     &testCertInfo{cert: testIBCAdminPK5, sk: testIBCAdminSK5},
		client:    &testCertInfo{cert: testIBCClientPK5, sk: testIBCClientSK5},
	},
}

func initIBCOrgMember(t *testing.T, info *orgMemberInfo) *orgMember {
	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := &test.GoLogger{}
	ibcProvider, err := newIBCACProvider(testIBCChainConfig, info.orgId, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, ibcProvider)

	testIBCProviderMsg(ibcProvider)

	localPrivKeyFile := filepath.Join(td, info.orgId+".key")
	localCertFile := filepath.Join(td, info.orgId+".crt")

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.consensus.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(info.consensus.cert), os.ModePerm)
	require.Nil(t, err)
	consensus, err := InitIBCSigningMember(testChainConfig, info.orgId, localPrivKeyFile, localCertFile)
	require.Nil(t, err)

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.admin.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(info.admin.cert), os.ModePerm)
	require.Nil(t, err)
	admin, err := InitIBCSigningMember(testChainConfig, info.orgId, localPrivKeyFile, localCertFile)
	require.Nil(t, err)

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.client.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(info.client.cert), os.ModePerm)
	require.Nil(t, err)
	client, err := InitIBCSigningMember(testChainConfig, info.orgId, localPrivKeyFile, localCertFile)
	require.Nil(t, err)
	return &orgMember{
		orgId:      info.orgId,
		acProvider: ibcProvider,
		consensus:  consensus,
		admin:      admin,
		client:     client,
	}
}

func testIBCProviderMsg(ip *ibcACProvider) {
	// CertFreeze
	certFreezeMsg := &msgbus.Message{Topic: msgbus.CertManageCertsFreeze, Payload: []string{testCAOrg4}}
	ip.OnMessage(certFreezeMsg)
	// CertsRevoke
	certRevokeMsg := &msgbus.Message{Topic: msgbus.CertManageCertsRevoke, Payload: []string{testTrustMember1}}
	ip.OnMessage(certRevokeMsg)
}
