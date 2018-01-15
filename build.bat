
call :build linux arm  
call :build linux amd64  
call :build linux 386  
call :build linux mips64le  
call :build linux mips64  
call :build darwin amd64  
call :build darwin 386   
call :build freebsd 386
call :build freebsd amd64
call :build windows 386 .exe
call :build windows amd64 .exe
call :armv5
goto :end

:build
  mkdir .\%1\%2
  set GOOS=%1
  set GOARCH=%2
  go build -o %1/%2/tcptunnel%3 -i -ldflags "-w -s" .
  goto :eof

:armv5
  mkdir .\linux\armv5
  set GOOS=linux
  set GOARCH=arm
  set GOARM=5
  go build -o linux/armv5/tcptunnel -i -ldflags "-w -s" .
  goto :eof

:end