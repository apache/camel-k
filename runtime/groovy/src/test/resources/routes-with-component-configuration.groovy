
component('seda') {
    queueSize 1234
}

from('timer:tick')
    .to('log:info')