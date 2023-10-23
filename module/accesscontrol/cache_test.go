package accesscontrol

import (
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	configPb "chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/protocol/v3/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

//var TestCert = ""
//var TestPK = ""

var memberCert = &acPb.Member{
	OrgId:      "org1",
	MemberType: acPb.MemberType_CERT,
	MemberInfo: []byte("-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDAwGbMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABORqoYNAw8ax\n9QOD94VaXq1dCHguarSKqAruEI39dRkm8Vu2gSHkeWlxzvSsVVqoN6ATObi2ZohY\nKYab2s+/QA2jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCDZOtAtHzfoZd/OQ2Jx5mIMgkqkMkH4SDvAt03yOrRnBzArBgNVHSME\nJDAigCA1JD9xHLm3xDUukx9wxXMx+XQJwtng+9/sHFBf2xCJZzAKBggqhkjOPQQD\nAgNIADBFAiEAiGjIB8Wb8mhI+ma4F3kCW/5QM6tlxiKIB5zTcO5E890CIBxWDICm\nAod1WZHJajgnDQ2zEcFF94aejR9dmGBB/P//\n-----END CERTIFICATE-----"),
}

var memberPK = &acPb.Member{
	OrgId:      "public",
	MemberType: acPb.MemberType_PUBLIC_KEY,
	MemberInfo: []byte("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA56Ts7nA8HqrApIkPFoHK\nNZSCo1SWxanjlkgBowLSjlatdYpTqKeE+mbNWFyl8R00JIuSsPf2pdIsdLhvNb6N\nL5uZ0bDlvaMv3Eg5q77Kt8TwJ12j6l3Gr8lrh7g8xYsIRbEUMjG0L/E4y4Fhlk7k\nDoGOrbiaA01vqlQDZVXCJCbK94oQOrokteMlyrl4/4bbilpWV8Sirc3mp12DMRPx\nGc3pGrGaxH8U263aHKFYj6+IKaPQ++RyL7L978fNCsnNuy8gnSynDMf1ddrGcIp0\nYIMXll3+58JO7EHvb2GQjhi6dPX057budvHfX3YJKFHnaDvXBBDCyV8V5lWrl5dV\n3QIDAQAB\n-----END PUBLIC KEY-----"),
}

func TestGetMemberPkAndAddress(t *testing.T) {

	chainConfig := &configPb.ChainConfig{
		Crypto: &configPb.CryptoConfig{
			Hash: "SHA256",
		},
		Vm: &configPb.Vm{
			AddrType: configPb.AddrType_ZXL,
		},
	}

	ctl := gomock.NewController(t)
	snapshot := mock.NewMockSnapshot(ctl)
	snapshot.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()

	tests := []struct {
		name        string
		member      *acPb.Member
		wantErr     bool
		wantPkPem   string
		wantAddress string
	}{
		{
			name:        "Test_Parse_Cert",
			member:      memberCert,
			wantErr:     false,
			wantPkPem:   "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5Gqhg0DDxrH1A4P3hVperV0IeC5q\ntIqoCu4Qjf11GSbxW7aBIeR5aXHO9KxVWqg3oBM5uLZmiFgphpvaz79ADQ==\n-----END PUBLIC KEY-----\n",
			wantAddress: "ZX3ff4d036ab551187d634501162b38153e299e5d9",
		},
		{
			name:        "Test_Parse_PK",
			member:      memberPK,
			wantErr:     false,
			wantPkPem:   "-----BEGIN PUBLIC KEY-----\nMIIBCgKCAQEA56Ts7nA8HqrApIkPFoHKNZSCo1SWxanjlkgBowLSjlatdYpTqKeE\n+mbNWFyl8R00JIuSsPf2pdIsdLhvNb6NL5uZ0bDlvaMv3Eg5q77Kt8TwJ12j6l3G\nr8lrh7g8xYsIRbEUMjG0L/E4y4Fhlk7kDoGOrbiaA01vqlQDZVXCJCbK94oQOrok\nteMlyrl4/4bbilpWV8Sirc3mp12DMRPxGc3pGrGaxH8U263aHKFYj6+IKaPQ++Ry\nL7L978fNCsnNuy8gnSynDMf1ddrGcIp0YIMXll3+58JO7EHvb2GQjhi6dPX057bu\ndvHfX3YJKFHnaDvXBBDCyV8V5lWrl5dV3QIDAQAB\n-----END PUBLIC KEY-----\n",
			wantAddress: "ZXcbee1a075ccc9b4ac668fb34dc90da160303df40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotPk, gotAddress, err := GetMemberPkAndAddress(tt.member, snapshot)

			if tt.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)

				gotPkPem, err := gotPk.String()
				require.Nil(t, err)

				require.Equal(t, tt.wantPkPem, gotPkPem)
				require.Equal(t, tt.wantAddress, gotAddress)
			}
		})
	}
}
