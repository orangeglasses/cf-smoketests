package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"fmt"
	"net/url"
)

func CreateOrGetUser(user ScimUser, jwtToken, authDomain string) (*ScimResource, *TestResult, *TestResult) {
	createUserResult := defaultTestResult()

	// Marshal user object to JSON bytes.
	userBytes, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	createUserBody := bytes.NewReader(userBytes)

	// Create request to create user.
	// https://docs.cloudfoundry.org/api/uaa/version/4.7.0/index.html#create-4
	createUserRequest, err := http.NewRequest(http.MethodPost, authDomain+"/Users", createUserBody)
	if err != nil {
		panic(err)
	}
	createUserRequest.Header.Add("Accept", "application/json")
	createUserRequest.Header.Add("Content-Type", "application/json")
	createUserRequest.Header.Add("Authorization", "Bearer "+jwtToken)

	httpClient := &http.Client{}
	createUserResponse, err := httpClient.Do(createUserRequest)
	if err != nil {
		panic(err)
	}
	defer createUserResponse.Body.Close()

	// Read create user response.
	responseBuffer := new(bytes.Buffer)
	responseBuffer.ReadFrom(createUserResponse.Body)

	// Check response status code.
	statusCode := createUserResponse.StatusCode
	if statusCode == http.StatusCreated {
		// User successfully created: get user id.
		var createdUser ScimResource
		err = json.Unmarshal(responseBuffer.Bytes(), &createdUser)
		if err != nil {
			panic(err)
		}
		return &createdUser, &createUserResult, nil
	}

	// User was not successfully created: figure out what went wrong.
	createUserResult.ParseErrorResponse(responseBuffer)

	// Check if user was already created.
	if statusCode == http.StatusConflict {
		// Conflict: user may already exist. Check error in response.
		if createUserResult.Error == "scim_resource_already_exists" {
			// This should not happen because user should be deleted by DeleteUser function at the end
			// of running all tests. However, we do not flag this as an error because user will probably
			// be deleted in next run.
			createUserResult.StatusCode = &statusCode

			// Attempt to get existing user.
			var retrievedUser *ScimResource
			retrievedUser, getUserResult := GetUserByUserName(jwtToken, authDomain, user.UserName)
			return retrievedUser, &createUserResult, &getUserResult
		}
	}

	// Return error: user was not created and user did not already exist.
	createUserResult.Result = false
	createUserResult.StatusCode = &statusCode

	return nil, &createUserResult, nil
}

func GetUserByUserName(jwtToken, authDomain, userName string) (*ScimResource, TestResult) {
	getUserResult := defaultTestResult()

	// Create request to retrieve specific user by user name.
	// https://docs.cloudfoundry.org/api/uaa/version/4.8.0/index.html#list-3
	filter := url.QueryEscape(fmt.Sprintf("userName eq \"%s\"", userName))
	getUsersRequest, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/Users?filter=%s", authDomain, filter), nil)
	if err != nil {
		panic(err)
	}
	getUsersRequest.Header.Add("Accept", "application/json")
	getUsersRequest.Header.Add("Authorization", "Bearer "+jwtToken)

	httpClient := &http.Client{}
	getUsersResponse, err := httpClient.Do(getUsersRequest)
	if err != nil {
		panic(err)
	}
	defer getUsersResponse.Body.Close()

	responseBuffer := new(bytes.Buffer)
	responseBuffer.ReadFrom(getUsersResponse.Body)
	statusCode := getUsersResponse.StatusCode

	if statusCode == http.StatusOK {
		var users ScimList
		if err = json.Unmarshal(responseBuffer.Bytes(), &users); err != nil {
			panic(err)
		}

		// Check if there are users with the specified user name.
		if users.TotalResults > 0 {
			userResource := users.Resources[0]
			return &userResource, getUserResult
		} else {
			getUserResult.Result = false
			getUserResult.StatusCode = &statusCode
			getUserResult.Error = "scim_resource_not_found"
			getUserResult.ErrorDescription = fmt.Sprintf("User %s does not exist", userName)
			return nil, getUserResult
		}
	}

	getUserResult.Result = false
	getUserResult.StatusCode = &statusCode
	getUserResult.ParseErrorResponse(responseBuffer)
	return nil, getUserResult
}

func GetGroups(jwtToken, authDomain string) ([]ScimResource, TestResult) {
	getGroupsResult := defaultTestResult()

	// Create request to retrieve all groups.
	// https://docs.cloudfoundry.org/api/uaa/version/4.7.0/index.html#list-3
	getGroupsRequest, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/Groups", authDomain), nil)
	if err != nil {
		panic(err)
	}
	getGroupsRequest.Header.Add("Accept", "application/json")
	getGroupsRequest.Header.Add("Authorization", "Bearer "+jwtToken)

	httpClient := &http.Client{}
	getGroupsResponse, err := httpClient.Do(getGroupsRequest)
	if err != nil {
		panic(err)
	}
	defer getGroupsResponse.Body.Close()

	responseBuffer := new(bytes.Buffer)
	responseBuffer.ReadFrom(getGroupsResponse.Body)
	statusCode := getGroupsResponse.StatusCode

	if statusCode == http.StatusOK {
		var groups ScimList
		if err = json.Unmarshal(responseBuffer.Bytes(), &groups); err != nil {
			panic(err)
		}
		return groups.Resources, getGroupsResult
	}

	getGroupsResult.Result = false
	getGroupsResult.StatusCode = &statusCode
	getGroupsResult.ParseErrorResponse(responseBuffer)
	return nil, getGroupsResult
}

func AddGroupMember(groupID, userID, jwtToken, authDomain string) TestResult {
	addGroupMemberResult := defaultTestResult()

	// Create request to add a member to a group.
	// https://docs.cloudfoundry.org/api/uaa/version/4.7.0/index.html#add-member
	user := make(map[string]string)
	user["origin"] = "uaa"
	user["type"] = "USER"
	user["value"] = userID
	userBytes, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	userReader := bytes.NewReader(userBytes)

	addGroupMemberRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/Groups/%s/members", authDomain, groupID), userReader)
	if err != nil {
		panic(err)
	}
	addGroupMemberRequest.Header.Add("Accept", "application/json")
	addGroupMemberRequest.Header.Add("Authorization", "Bearer "+jwtToken)
	addGroupMemberRequest.Header.Add("Content-Type", "application/json")

	// Perform request.
	httpClient := &http.Client{}
	addGroupMemberResponse, err := httpClient.Do(addGroupMemberRequest)
	if err != nil {
		panic(err)
	}
	defer addGroupMemberResponse.Body.Close()

	// Check response.
	responseBuffer := new(bytes.Buffer)
	responseBuffer.ReadFrom(addGroupMemberResponse.Body)
	statusCode := addGroupMemberResponse.StatusCode
	if statusCode == http.StatusCreated {
		return addGroupMemberResult
	}

	// Member may have already been added to group in the case there was a failure deleting the user in a previous run.
	addGroupMemberResult.ParseErrorResponse(responseBuffer)
	if statusCode == http.StatusConflict && addGroupMemberResult.Error == "member_already_exists" {
		addGroupMemberResult.StatusCode = &statusCode
		return addGroupMemberResult
	}

	addGroupMemberResult.Result = false
	addGroupMemberResult.StatusCode = &statusCode
	return addGroupMemberResult
}

func DeleteUser(userID, jwtToken, authDomain string) TestResult {
	deleteUserTestResult := defaultTestResult()

	// Create request to delete user.
	// https://docs.cloudfoundry.org/api/uaa/version/4.7.0/index.html#delete-3
	userDeleteRequest, err := http.NewRequest(http.MethodDelete, authDomain+"/Users/"+userID, nil)
	if err != nil {
		panic(err)
	}
	userDeleteRequest.Header.Add("Accept", "application/json")
	userDeleteRequest.Header.Add("Content-Type", "application/json")
	userDeleteRequest.Header.Add("Authorization", "Bearer "+jwtToken)

	httpClient := &http.Client{}
	userDeleteResponse, err := httpClient.Do(userDeleteRequest)
	if err != nil {
		panic(err)
	}
	defer userDeleteResponse.Body.Close()

	// Check response.
	statusCode := userDeleteResponse.StatusCode
	if statusCode != http.StatusOK {
		deleteUserTestResult.Result = false
		deleteUserTestResult.StatusCode = &statusCode

		// Try parse error response.
		responseBuffer := new(bytes.Buffer)
		responseBuffer.ReadFrom(userDeleteResponse.Body)
		deleteUserTestResult.ParseErrorResponse(responseBuffer)
	}
	return deleteUserTestResult
}
