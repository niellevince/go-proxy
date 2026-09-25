# Client samples

Call the published hostname. The URL host is the `from` value in `proxies.json`. Send the route key in `X-PROXY-KEY`. The proxy removes that header before the request reaches the upstream.

Replace `https://hello.world.com/path` and `CHANGE_ME` with your host and key. For a public route (`key` is empty), omit the header.

## fetch

```javascript
const response = await fetch("https://hello.world.com/path", {
  headers: {
    "X-PROXY-KEY": "CHANGE_ME",
  },
});
const body = await response.text();
```

## axios

```javascript
import axios from "axios";

const response = await axios.get("https://hello.world.com/path", {
  headers: {
    "X-PROXY-KEY": "CHANGE_ME",
  },
});
```

## curl

```bash
curl -H "X-PROXY-KEY: CHANGE_ME" https://hello.world.com/path
```

## Python

```python
import requests

response = requests.get(
    "https://hello.world.com/path",
    headers={"X-PROXY-KEY": "CHANGE_ME"},
)
print(response.text)
```

## Go

```go
req, err := http.NewRequest(http.MethodGet, "https://hello.world.com/path", nil)
if err != nil {
    return err
}
req.Header.Set("X-PROXY-KEY", "CHANGE_ME")

resp, err := http.DefaultClient.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()
```

## PHP

```php
$ch = curl_init("https://hello.world.com/path");
curl_setopt($ch, CURLOPT_HTTPHEADER, ["X-PROXY-KEY: CHANGE_ME"]);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$body = curl_exec($ch);
curl_close($ch);
```
