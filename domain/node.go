package domain

const UserLabel = "User"
const RepoLabel = "Repo"
const ContributorLabel = ":CONTRIBUTOR"
const OwnerLabel = ":OWNS"
const FollowerLabel = ":FOLLOWS"

type RelationshipNodeData struct {
	Label      string
	MatchProp  string
	ParamName  string
	ParamValue string
}
