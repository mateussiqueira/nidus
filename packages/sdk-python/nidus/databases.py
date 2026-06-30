class DatabasesAPI:
    def __init__(self, client):
        self._client = client

    def list(self):
        return self._client._request("GET", "/databases")

    def create(self, name, project_id):
        return self._client._request("POST", "/databases", {
            "name": name,
            "project_id": project_id,
        })
