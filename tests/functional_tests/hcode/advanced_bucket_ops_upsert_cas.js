function OnUpdate(doc, meta) {
  var {meta} = couchbase.get(dst_bucket, meta);

  // Change the cas
  dst_bucket[meta.id] = 'Changed';
  var {success, error} = couchbase.upsert(dst_bucket, meta, 'cas mis match');
  if(!success && error.cas_mismatch && error.name === 'LCB_KEY_EEXISTS') {
    var {meta} = couchbase.get(dst_bucket, meta);

    var {success} = couchbase.upsert(dst_bucket, meta, 'success');
    if(success) {
      delete dst_bucket[meta.id];
    }
  }
}
