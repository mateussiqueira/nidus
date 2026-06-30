class DomainsAPI:
    def __init__(self, client):
        self._client = client

    def list(self, project_id):
        return self._client._request("GET", f"/projects/{project_id}/domains")

    def add(self, project_id, domain):
        return self._client._request("POST", f"/projects/{project_id}/domains", {
            "domain": domain,
        })

    def delete(self, project_id, domain_id):
        return self._client._request(
            "DELETE", f"/projects/{project_id}/domains/{domain_id}"
        )
