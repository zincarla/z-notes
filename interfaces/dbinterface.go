package interfaces

//DBInterface is a generic interface to allow swappable databases
type DBInterface interface {
	////Account operations
	//CreateUser is used to create and add a user to the AuthN database (return new userid and nil on success)
	CreateUser(userData UserInformation) (uint64, error)
	//RemoveUser Removes a user from the user database (nil on success)
	RemoveUser(userID uint64) error
	//SetUserDisableState disables or enables a user account
	SetUserDisableState(userID uint64, isDisabled bool) error
	//UpdateUserNameEmail updates a user based on DBID to have the name/email located in the userData object. If an email has not been verified, it will be silently ignored
	UpdateUserNameEmail(userData UserInformation) error
	//GetUser returns a completed UserInformation object for the user specified, OIDCIssuer and Subject must be specified, or the DBID
	GetUser(userData UserInformation) (UserInformation, error)

	////Page operations
	//CreatePage is used to create a new page (return new pageid and nil on success)
	CreatePage(pageData Page) (uint64, error)
	//UpdatePage updates a page
	UpdatePage(pageData Page) error
	//RemovePage removes a page (error nil on success)
	RemovePage(pageID uint64) error
	//GetPage returns a page's data
	GetPage(pageID uint64) (Page, error)
	//GetPageChildren returns incomplete page data for children of the specified page (Content not included)
	GetPageChildren(pageID uint64) ([]Page, error)
	//GetPagePath returns a slice representing the page to the root, order of slice is determined by rootFirst
	GetPagePath(pageID uint64, rootFirst bool) ([]Page, error)
	//GetRootPages returns incomplete page data for root pages of the specified user (Content not included)
	GetRootPages(userID uint64) ([]Page, error)
	//SearchPages returns incomplete page data for for pages that match the supplied query
	SearchPages(userID uint64, query string, limit uint64, offset uint64) ([]Page, error)
	//GetPageRevisions returns a slice of page revisions given a pageID, the total revisions
	GetPageRevisions(pageID uint64, limit uint64, offset uint64) ([]Page, uint64, error)
	//GetPageRevision returns specific page revision (Incomplete as revisions only contain partial information)
	GetPageRevision(pageID uint64, revisionID uint64) (Page, error)

	////PagePermissions
	//UpdatePermission creates or updates a pagepermission
	UpdatePermission(permission UserPageAccess) error
	//RemovePermission removes a PagePermission (error nil on success)
	RemovePermission(permissionID uint64) error
	//GetPermissions returns the permissions assigned directly to a page with the given id
	GetPermissions(pageID uint64) ([]UserPageAccess, error)
	//GetPermission returns the permission assigned directly to a page for a user
	GetPermission(pageAccess UserPageAccess) (UserPageAccess, error)
	//GetEffectivePermission returns the effective permissions for a user on a page, this takes into account inherited permissions
	GetEffectivePermission(pageAccess UserPageAccess) (UserPageAccess, error)

	////Tokens
	//CreateToken creates a new token owned by the specified ownerID, returns APITokenInformation, and/or an error
	CreateToken(tokenInfo APITokenInformation) (APITokenInformation, error)
	//GetToken returns a token based on FriendlyID
	GetToken(tokenID string) (APITokenInformation, error)
	//GetTokenByID returns a token based on ID
	GetTokenByID(tokenID uint64) (APITokenInformation, error)
	//GetTokens returns a slice of tokens based on UserID
	GetTokens(userID uint64) ([]APITokenInformation, error)
	//RefreshToken refreshes a token by crating a new tokenFriendlyID returns the new tokenFriendlyID, and/or an error
	RefreshToken(tokenInfo APITokenInformation) (APITokenInformation, error)
	//RemoveToken deletes a token from the database
	RemoveToken(tokenFriendlyID string) error
	////TokenPermissions
	//UpdateTokenPermission creates or updates a pagepermission for tokens
	UpdateTokenPermission(permission TokenPageAccess) error
	//RemoveTokenPermission removes a PagePermission for a token (error nil on success)
	RemoveTokenPermission(permissionID uint64) error
	//GetTokenPermissions returns the token permissions assigned directly to a page with the given id
	GetTokenPermissions(pageID uint64) ([]TokenPageAccess, error)
	//GetTokenPermission returns the token permission assigned directly to a page
	GetTokenPermission(pageAccess TokenPageAccess) (TokenPageAccess, error)
	//GetEffectiveTokenPermission returns the effective permissions for a token on a page, this takes into account inherited permissions
	GetEffectiveTokenPermission(pageAccess TokenPageAccess) (TokenPageAccess, error)

	//Maitenance
	//InitDatabase connects to a database, and if needed, creates and or updates tables
	InitDatabase() error
}

const (
	//AnonymousUserID Database ID of the Anonymous user
	AnonymousUserID uint64 = 1
	//AuthenticatedUserID Database ID of the Authenticated user
	AuthenticatedUserID uint64 = 2
)
