class DeploymentsAPI:
    def __init__(self, client):
        self._client = client

    def list(self, project_id):
        return self._client._request("GET", f"/projects/{project_id}/deployments")

    def logs(self, project_id, deployment_id):
        return self._client._request(
            "GET", f"/projects/{project_id}/deployments/{deployment_id}/logs"
        )
