import requests

from stackrun.auth import AuthAPI
from stackrun.projects import ProjectsAPI
from stackrun.deployments import DeploymentsAPI
from stackrun.domains import DomainsAPI
from stackrun.databases import DatabasesAPI


class NidusClient:
    def __init__(self, api_url="https://api.nidus.app", token=None):
        self.api_url = api_url.rstrip("/")
        self.token = token
        self._session = requests.Session()
        self._session.headers.update({
            "Content-Type": "application/json",
            "Accept": "application/json",
        })
        if token:
            self.set_token(token)

    def set_token(self, token):
        self.token = token
        self._session.headers.update({"Authorization": f"Bearer {token}"})

    def _request(self, method, path, data=None):
        url = f"{self.api_url}{path}"
        response = self._session.request(method, url, json=data)
        response.raise_for_status()
        return response.json()

    @property
    def auth(self):
        return AuthAPI(self)

    @property
    def projects(self):
        return ProjectsAPI(self)

    @property
    def deployments(self):
        return DeploymentsAPI(self)

    @property
    def domains(self):
        return DomainsAPI(self)

    @property
    def databases(self):
        return DatabasesAPI(self)
