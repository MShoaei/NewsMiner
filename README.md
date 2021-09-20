# NewsMiner

NewsMiner is a web crawler to extract data from Persian news websites.

## Database

use `init-db` to initialize the database

## Running

to start extracting from all websites run the command below:

```bash
go run ./main.go -d "NewsMiner" -c "mongodb://localhost:27017/?compressors=disabled&gssapiServiceName=mongodb" -p 10 -t 10 --farsnews --isna --tabnak --tasnim --yjc
```
