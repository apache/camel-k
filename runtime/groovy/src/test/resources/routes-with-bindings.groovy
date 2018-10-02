
context {
    registry {
        bind 'myEntry1', 'myRegistryEntry1'
        bind 'myEntry2', 'myRegistryEntry2'
    }
}

from('timer:tick')
    .to('log:info')