# asynchronous HTTP request
from pyodide.http import open_url
import json

url = "https://gitlab.com/api/v4/projects"
response = open_url(url)
data = json.load(response)
print(data)
