# NewsMiner

NewsMiner is a web crawler to extract data from Persian news websites.

## Database

```bash
mkdir data
docker run -d --name NewsMinerdb \
    -p 27017:27017 -v $(pwd)/data:/data/db \
    mongo:4
```

to connect:

```bash
mongo "localhost/News" \
--authenticationDatabase "admin" \
--username "miner" \
--password "password"
```

extract data as json:

```bash
mongoexport --uri="mongodb://miner:password@localhost:27017/News?authSource=admin" \
--collection="data" \
--out="data.json"
```
