# asynchronous HTTP request
from pyodide.http import pyfetch

url = "https://gitlab.com/api/v4/projects"
response = await pyfetch(url)
data = await response.json()
print(data)
