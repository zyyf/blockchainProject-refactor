@echo off
for /L %%i in (1,1,9) do start client.exe -p 1000%%i -s 192.168.1.8:30000
for /L %%i in (10,1,16) do start client.exe -p 100%%i -s 192.168.1.8:30000