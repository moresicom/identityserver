package organization

import (
	"gopkg.in/validator.v2"
	"regexp"
)

type Organization struct {
	DNS             []string        `json:"dns"`
	Globalid        string          `json:"globalid"`
	Members         []string        `json:"members"`
	Owners          []string        `json:"owners" validate:"min=1"`
	PublicKeys      []string        `json:"publicKeys"`
	SecondsValidity int             `json:"secondsvalidity"`
	OrgOwners       []string        `json:"orgowners"`  //OrgOwners are other organizations that are owner of this organization
	OrgMembers      []string        `json:"orgmembers"` //OrgMembers are other organizations that are member of this organization
	RequiredScopes  []RequiredScope `json:"requiredscopes"`
}

// IsValid performs basic validation on the content of an organizations fields
func (org *Organization) IsValid() bool {
	regex, _ := regexp.Compile(`^[a-z\d\-_\s]{3,150}$`)
	return validator.Validate(org) == nil && regex.MatchString(org.Globalid)
}

func (org *Organization) IsValidSubOrganization() bool {
	regex, _ := regexp.Compile(`^[a-z\d\-_\s\.]{3,150}$`)
	return validator.Validate(org) == nil && regex.MatchString(org.Globalid)
}
