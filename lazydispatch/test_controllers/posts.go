package test_controllers

type PostsController struct {
}

func (p *PostsController) Index() string {
	return "index"
}
func (p *PostsController) Show() string {
	return "show"
}
func (p *PostsController) New() string {
	return "new"
}
func (p *PostsController) Create() string {
	return "create"
}
func (p *PostsController) Edit() string {
	return "edit"
}
func (p *PostsController) Update() string {
	return "update"
}
func (p *PostsController) Delete() string {
	return "delete"
}
func (p *PostsController) Search() string {
	return "search"
}
func (p *PostsController) POSTSearch() string {
	return "post_search"
}
func (p *PostsController) MemberApprove() string {
	return "member_approve"
}
func (p *PostsController) MemberPUT_Reject() string {
	return "member_put_reject"
}
