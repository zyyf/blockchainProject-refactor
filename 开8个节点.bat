@echo off
for /L %%i in (1,1,8) do start client.exe -p 1100%%i -local -s 127.0.0.1:30000
