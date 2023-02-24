# hz.tools/example/fm - listen to FM radio using an SDR

Example usage with an Ettus USRP UHD Radio, like an Ettus B210

```bash
$ go build .
$ ./fm --sdr=uhd --frequency=88.5MHz --gains=RX0PGA=60
```
