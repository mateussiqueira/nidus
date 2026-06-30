use crate::project_service_server::{ProjectService, ProjectServiceServer};
use crate::{Project, GetProjectRequest, ListProjectsRequest, ListProjectsResponse, ResolveSlugRequest, ResolveSlugResponse};
use tonic::{Request, Response, Status};

pub struct StackRunProjectService;

#[tonic::async_trait]
impl ProjectService for StackRunProjectService {
    async fn get_project(&self, req: Request<GetProjectRequest>) -> Result<Response<Project>, Status> {
        Ok(Response::new(Project {
            id: req.into_inner().id,
            name: "example".into(),
            slug: "example".into(),
            status: "ACTIVE".into(),
            port: 3000,
            ..Default::default()
        }))
    }

    async fn resolve_slug(&self, req: Request<ResolveSlugRequest>) -> Result<Response<ResolveSlugResponse>, Status> {
        let _slug = req.into_inner().slug;
        Ok(Response::new(ResolveSlugResponse {
            port: 8095,
            status: "ACTIVE".into(),
        }))
    }

    async fn list_projects(&self, _req: Request<ListProjectsRequest>) -> Result<Response<ListProjectsResponse>, Status> {
        Ok(Response::new(ListProjectsResponse { projects: vec![] }))
    }
}

pub fn create_project_server() -> ProjectServiceServer<StackRunProjectService> {
    ProjectServiceServer::new(StackRunProjectService)
}
