function MessagePort() {
  this._queue = []
  this._otherPort = null
  this.onmessage = null
}

MessagePort.prototype.postMessage = function (msg) {
  if (!this._otherPort) {
    this._queue.push(msg)
    return
  }

  const deliverMessage = () => {
    if (this._otherPort.onmessage) {
      this._otherPort.onmessage({ data: msg })
    }
  }

  if (typeof queueMicrotask === "function") {
    queueMicrotask(deliverMessage)
  } else {
    setTimeout(deliverMessage, 0)
  }
}

MessagePort.prototype._connect = function (otherPort) {
  this._otherPort = otherPort
  while (this._queue.length > 0) {
    this.postMessage(this._queue.shift())
  }
}

function MessageChannel() {
  this.port1 = new MessagePort()
  this.port2 = new MessagePort()
  this.port1._connect(this.port2)
  this.port2._connect(this.port1)
}

export { MessageChannel }

function TextEncoder() {}

TextEncoder.prototype.encode = function (string) {
  var octets = []
  var length = string.length
  var i = 0
  while (i < length) {
    var codePoint = string.codePointAt(i)
    var c = 0
    var bits = 0
    if (codePoint <= 0x0000007f) {
      c = 0
      bits = 0x00
    } else if (codePoint <= 0x000007ff) {
      c = 6
      bits = 0xc0
    } else if (codePoint <= 0x0000ffff) {
      c = 12
      bits = 0xe0
    } else if (codePoint <= 0x001fffff) {
      c = 18
      bits = 0xf0
    }
    octets.push(bits | (codePoint >> c))
    c -= 6
    while (c >= 0) {
      octets.push(0x80 | ((codePoint >> c) & 0x3f))
      c -= 6
    }
    i += codePoint >= 0x10000 ? 2 : 1
  }
  return octets
}

export { TextEncoder }
