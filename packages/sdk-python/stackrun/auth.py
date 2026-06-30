class AuthAPI:
    def __init__(self, client):
        self._client = client

    def login(self, email, password):
        data = self._client._request("POST", "/auth/login", {
            "email": email,
            "password": password,
        })
        if "token" in data:
            self._client.set_token(data["token"])
        return data

    def register(self, email, password, name):
        data = self._client._request("POST", "/auth/register", {
            "email": email,
            "password": password,
            "name": name,
        })
        if "token" in data:
            self._client.set_token(data["token"])
        return data
