package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
)

const (
	uaaSmokeUsername = "smokeuser"
	uaaSmokePassword = "smokepassword"
	smokeScope       = "smoketest.extinguish"

	ssoKey  = "sso"
	ssoName = "Single Sign-On"

	ssoTestBinding           = "Read service binding"
	ssoErrorBinding          = "Service p-identity not or incorrectly configured in VCAP_SERVICES"
	ssoTestClientCredentials = "Client credentials grant"
	ssoTestCreateUser        = "Create local user"
	ssoTestGetUser           = "Get local user"
	ssoTestGetGroups         = "Get local groups/scopes"
	ssoTestAddGroupMember    = "Add local user to group"
	ssoTestPassword          = "Resource owner password credentials grant"
	ssoTestAuthCodeUAA       = "Authorization code grant (UAA)"
	ssoTestAuthCodeADFS      = "Authorization code grant (ADFS)"
	ssoTestDeleteUser        = "Delete local user"
)

// TODO: find a way to externalize these (they don't come from the VCAP_SERVICES, perhaps in the Concourse pipeline?).
const (
	adfsSmokeUsername = "AD\\SomeAccountName"
	adfsSmokePassword = "SomePassword"
)

var (
	uaaResourceUrl  string
	adfsResourceUrl string
)

type ssoTest struct {
	authDomain   string
	clientId     string
	clientSecret string
}

func ssoTestNew(env *cfenv.App) SmokeTest {
	adfsResourceUrl = os.Getenv("ADFS_RES_URL")
	uaaResourceUrl = os.Getenv("UAA_RES_URL")

	if uaaResourceUrl == "" || adfsResourceUrl == "" {
		return nil
	}

	identityServices, err := env.Services.WithLabel("p-identity")
	if err != nil {
		return &ssoTest{"", "", ""}
	}

	creds := identityServices[0].Credentials
	return &ssoTest{
		creds["auth_domain"].(string),
		creds["client_id"].(string),
		creds["client_secret"].(string),
	}

}

func (t *ssoTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)
	oauth2FlowsTestResult := t.internalRun()

	// Transform test results into SmokeTestResult structure.
	if oauth2FlowsTestResult.ServiceBindingError {
		results = append(results, SmokeTestResult{Name: ssoTestBinding, Result: false, Error: ssoErrorBinding})
		return OverallResult(ssoKey, ssoName, results)
	}
	results = append(results, SmokeTestResult{Name: ssoTestBinding, Result: true})

	clientCredentials := oauth2FlowsTestResult.ClientCredentials
	results = append(results, SmokeTestResult{Name: ssoTestClientCredentials, Result: clientCredentials.Result, Error: clientCredentials.Error, ErrorDescription: clientCredentials.ErrorDescription, StatusCode: clientCredentials.StatusCode})
	if clientCredentials.HasError() {
		return OverallResult(ssoKey, ssoName, results)
	}

	createUser := oauth2FlowsTestResult.CreateUser
	results = append(results, SmokeTestResult{Name: ssoTestCreateUser, Result: createUser.Result, Error: createUser.Error, ErrorDescription: createUser.ErrorDescription, StatusCode: createUser.StatusCode})
	if createUser.HasError() {
		return OverallResult(ssoKey, ssoName, results)
	}

	if getUser := oauth2FlowsTestResult.GetUser; getUser != nil {
		results = append(results, SmokeTestResult{Name: ssoTestGetUser, Result: getUser.Result, Error: getUser.Error, ErrorDescription: getUser.ErrorDescription, StatusCode: getUser.StatusCode})
		if getUser.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	if getGroups := oauth2FlowsTestResult.GetGroups; getGroups != nil {
		results = append(results, SmokeTestResult{Name: ssoTestGetGroups, Result: getGroups.Result, Error: getGroups.Error, ErrorDescription: getGroups.ErrorDescription, StatusCode: getGroups.StatusCode})
		if getGroups.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	if addGroupMember := oauth2FlowsTestResult.AddGroupMember; addGroupMember != nil {
		results = append(results, SmokeTestResult{Name: ssoTestAddGroupMember, Result: addGroupMember.Result, Error: addGroupMember.Error, ErrorDescription: addGroupMember.ErrorDescription, StatusCode: addGroupMember.StatusCode})
		if addGroupMember.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	if passwordGrant := oauth2FlowsTestResult.Password; passwordGrant != nil {
		results = append(results, SmokeTestResult{Name: ssoTestPassword, Result: passwordGrant.Result, Error: passwordGrant.Error, ErrorDescription: passwordGrant.ErrorDescription, StatusCode: passwordGrant.StatusCode})
		if passwordGrant.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	if adfsAuthCode := oauth2FlowsTestResult.AuthorizationCodeADFS; adfsAuthCode != nil {
		results = append(results, SmokeTestResult{Name: ssoTestAuthCodeADFS, Result: adfsAuthCode.Result, Error: adfsAuthCode.Error, ErrorDescription: adfsAuthCode.ErrorDescription, StatusCode: adfsAuthCode.StatusCode})
		if adfsAuthCode.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	/*	if uaaAuthCode := oauth2FlowsTestResult.AuthorizationCodeUAA; uaaAuthCode != nil {
			results = append(results, SmokeTestResult{Name: ssoTestAuthCodeUAA, Result: uaaAuthCode.Result, Error: uaaAuthCode.Error, ErrorDescription: uaaAuthCode.ErrorDescription, StatusCode: uaaAuthCode.StatusCode})
			if uaaAuthCode.HasError() {
				return OverallResult(ssoKey, ssoName, results)
			}
		}
	*/
	fmt.Printf("DeleteUser: %v\n", oauth2FlowsTestResult.DeleteUser)
	if deleteUser := oauth2FlowsTestResult.DeleteUser; deleteUser != nil {
		results = append(results, SmokeTestResult{Name: ssoTestDeleteUser, Result: deleteUser.Result, Error: deleteUser.Error, ErrorDescription: deleteUser.ErrorDescription, StatusCode: deleteUser.StatusCode})
		if deleteUser.HasError() {
			return OverallResult(ssoKey, ssoName, results)
		}
	}

	return OverallResult(ssoKey, ssoName, results)
}

func (t *ssoTest) internalRun() Oauth2FlowsTestResult {
	oauth2FlowsTestResult := &Oauth2FlowsTestResult{}

	fmt.Println("Found client_id: " + t.clientId)
	if t.clientId == "" {
		fmt.Println("No client_id found")
		oauth2FlowsTestResult.ServiceBindingError = true
		return *oauth2FlowsTestResult
	}

	// Authenticate against UAA using client_credentials grant type and provided client id and secret.
	clientCredentialsTokenResponse, clientCredentialsTestResult := ClientCredentialsAuthentication(t.clientId, t.clientSecret, t.authDomain)
	oauth2FlowsTestResult.ClientCredentials = &clientCredentialsTestResult
	if clientCredentialsTestResult.HasError() {
		return *oauth2FlowsTestResult
	}

	// Create a local user, authenticating with the token we acquired above (which should have scim.write scope).
	// SCIM stands for System for Cross-domain Identity Management (http://www.simplecloud.info/).
	user := ScimUser{
		UserName:     uaaSmokeUsername,
		Name:         ScimUserName{Formatted: "Smoke User", FamilyName: "User", GivenName: "Smoke"},
		Emails:       []ScimAttribute{{Value: "smokeuser@smoke.itq.nl"}},
		Active:       true,
		Verified:     true,
		Origin:       "uaa",
		Password:     uaaSmokePassword,
		ScimResource: ScimResource{ExternalID: "", Meta: nil, Scim: Scim{Schemas: []string{"urn:scim:schemas:core:1.0"}}},
	}
	createdUser, createUserTestResult, getUserTestResult := CreateOrGetUser(user, clientCredentialsTokenResponse.AccessToken, t.authDomain)
	oauth2FlowsTestResult.CreateUser = createUserTestResult
	oauth2FlowsTestResult.GetUser = getUserTestResult
	if createUserTestResult.HasError() || (getUserTestResult != nil && getUserTestResult.HasError()) {
		return *oauth2FlowsTestResult
	}

	if createdUser != nil {
		// Delete local user after we're finished (via defer).
		defer func(res *Oauth2FlowsTestResult) {
			deleteUserTestResult := DeleteUser(createdUser.ID, clientCredentialsTokenResponse.AccessToken, t.authDomain)
			fmt.Printf("Delete user: %v\n", deleteUserTestResult)
			res.DeleteUser = &deleteUserTestResult
		}(oauth2FlowsTestResult)

		// Get all groups (to be able to assign new user to groups).
		groups, getGroupsResult := GetGroups(clientCredentialsTokenResponse.AccessToken, t.authDomain)
		oauth2FlowsTestResult.GetGroups = &getGroupsResult
		if getGroupsResult.HasError() {
			return *oauth2FlowsTestResult
		}

		// Get smoketest.extinguish group.
		var smokeExtinguishGroup ScimResource
		for i := range groups {
			if groups[i].DisplayName == smokeScope {
				smokeExtinguishGroup = groups[i]
				break
			}
		}

		// Assign user to smoketest.extinguish group.
		addMemberResult := AddGroupMember(smokeExtinguishGroup.ID, createdUser.ID, clientCredentialsTokenResponse.AccessToken, t.authDomain)
		oauth2FlowsTestResult.AddGroupMember = &addMemberResult
		if addMemberResult.HasError() {
			return *oauth2FlowsTestResult
		}

		// Authenticate directly against UAA with newly created user using password grant type.
		// (https://tools.ietf.org/html/rfc6749#section-4.3)
		// This does not involve ADFS yet, goes directly to UAA.
		_, userTokenTestResult := PasswordAuthentication(t.clientId, t.clientSecret, t.authDomain, uaaSmokeUsername, uaaSmokePassword)
		oauth2FlowsTestResult.Password = &userTokenTestResult
		if userTokenTestResult.HasError() {
			return *oauth2FlowsTestResult
		}

		// Authenticate against ADFS using the authorization code grant type (https://tools.ietf.org/html/rfc6749#section-4.1).
		_, adfsAuthorizationCodeResult := AdfsAuthorizationCodeAuthentication(adfsSmokeUsername, adfsSmokePassword)
		oauth2FlowsTestResult.AuthorizationCodeADFS = &adfsAuthorizationCodeResult
		if adfsAuthorizationCodeResult.HasError() {
			return *oauth2FlowsTestResult
		}

		/*
			// Authenticate against UAA using the authorization code grant type (https://tools.ietf.org/html/rfc6749#section-4.1).
			// Does still not involve ADFS yet. This requires an application that is protected by a UAA client.
			_, uaaAuthorizationCodeResult := UaaAuthorizationCodeAuthentication(uaaSmokeUsername, uaaSmokePassword)
			oauth2FlowsTestResult.AuthorizationCodeUAA = &uaaAuthorizationCodeResult
			if uaaAuthorizationCodeResult.HasError() {
				return *oauth2FlowsTestResult
			}
		*/
	}

	return *oauth2FlowsTestResult
}

type Oauth2FlowsTestResult struct {
	ServiceBindingError   bool        `json:"serviceBindingError"`
	ClientCredentials     *TestResult `json:"clientCredentials,omitempty"`
	CreateUser            *TestResult `json:"createUser,omitempty"`
	GetUser               *TestResult `json:"getUser,omitempty"`
	GetGroups             *TestResult `json:"getGroups,omitempty"`
	AddGroupMember        *TestResult `json:"addGroupMemberResult,omitempty"`
	Password              *TestResult `json:"password,omitempty"`
	AuthorizationCodeUAA  *TestResult `json:"authCodeUAA,omitempty"`
	AuthorizationCodeADFS *TestResult `json:"authCodeAdfs,omitempty"`
	DeleteUser            *TestResult `json:"deleteUser,omitempty"`
}
