function OnUpdate(doc, meta) {
    try{
        var response = curl('HEAD', localhost);
        log(response);
        if (!verifyResponse(response)) {
            throw 'inconsistent response';
        }
        dst_bucket[meta.id] = JSON.stringify(response);
    } catch(e) {
        log('error', e);
    }
}

function verifyResponse(response) {
    var expected = {
        status : 200
    };

    if(response.status !== expected.status) {
        return false;
    }
    return true;
}