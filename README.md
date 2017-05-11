# Trail
A simple REST API for tracking Interruptions

  If you're anything like me, you'll work along on something with much passion and then you're interrupted by something else... An IM, A system page, someone behind you asking questions, a phone call, etc. This API can be used to track those interruptions for better time management. 
  
  Start it up via Docker:
  
  ```$ docker build . -t trail
  $ docker run -d -p 8080:8080 trail <parameters>
  ``` 
  Where <parameters> can include the following:
  * -port [webPort] (default 8080)
  * -host [mongoHost]:[mongoPort] 
  * -user [mongoUser]
  * -pass [mongoPassword] 
  * -db [mongoDatabase]
  
  To use it from a curl, try the following:
  
  ```$ curl -XPOST localhost:8080/int \ 
  -H "Content-Type: application/json" \
  -d '{"what":"Finish making interruption API","method":0}'
  ```
  
  Then later see a list of your interruptions with:
  
  ```$ curl localhost:8080/int```
