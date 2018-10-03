
rest {
    configuration {
        host 'my-host'
        port '9192'
    }

    configuration('undertow') {
        host 'my-undertow-host'
        port '9193'
    }

    path('/my/path') {

    }
}

from('timer:tick')
    .to('log:info')