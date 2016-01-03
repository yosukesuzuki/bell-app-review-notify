/**
 * Created by suzukiyosuke on 11/23/15.
 */

var assert = require('assert');

describe('patch api tests', function() {
    it('after login', function(done) {
        browser
            .url('/')
            .getText('h3').then(function(value){
            assert.equal(value, 'Bell Apps');

            })
            .call(done);
    });
});
