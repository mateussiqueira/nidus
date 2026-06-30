class ProjectsAPI:
    def __init__(self, client):
        self._client = client

    def list(self):
        return self._client._request("GET", "/projects")

    def get(self, id):
        return self._client._request("GET", f"/projects/{id}")

    def create(self, data):
        return self._client._request("POST", "/projects", data)

    def deploy(self, id, branch=None):
        payload = {}
        if branch is not None:
            payload["branch"] = branch
        return self._client._request("POST", f"/projects/{id}/deploy", payload or None)

    def envs(self, id):
        return self._client._request("GET", f"/projects/{id}/envs")

    def env_set(self, id, key, value):
        return self._client._request("PATCH", f"/projects/{id}/envs", {
            "key": key,
            "value": value,
        })

    def volumes(self, id):
        return self._client._request("GET", f"/projects/{id}/volumes")

    def volume_create(self, id, name, mount_path):
        return self._client._request("POST", f"/projects/{id}/volumes", {
            "name": name,
            "mount_path": mount_path,
        })
