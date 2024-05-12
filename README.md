# Simple one-shot RSS aggregator
**❗For now, only available with installed Golang on the system**<br/>
**❗Not optimized for best performance**

### Steps
1. Make new directory
2. Place main.go inside created directory
3. Open any type of console
4. Navigate to the directory with main.go
5. Type ```go run main.go```

### Output
User interface will guide you through the process.

**guids.json** stores all parsed guid values from guid tags on XML recieved via Get request. Next time you run the program, after parsing step, it will check all **guids** against **newly parsed XML** and creates new **data_** file with only items that are new.<br/>

> ~~*Eventually, I will add support for SQLite3 database to drop the usage of JSON*~~

> *Eventually, I will add support to clean old entries*

**data_** file contains **date** and **time** in the filename. The timestamp is the program start time.<br/>

### Program is "One-Shot"
There is **no server** that updates in the background, for now. I don't like the idea of constantly running server, for now. <br/>

### How to add link
Open **main.go** in any text editor you want. Find first or last appearance of `newLinks.Push`. You can copy and paste an entire line, changing link address and giving it custom name as the first *string* parameter.

> *Later, I will make it as package and expose API, so you could use it inside your program. You need no fear, there will be no JSON strings, just methods to operate on parsed struct*

## P.S.

> [!IMPORTANT]
> Feel free to contribute or use.