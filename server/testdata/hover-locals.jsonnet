local test = {
  local targets = [
    i
    for i in std.objectFields({ test: 'test' })
  ],
};

local get_sentinel_id(i) =
  local hash = std.md5('%s\nindex: %d' % ['redis-master.svc.cluster.local', i]);
  std.objectFields('');

{}
