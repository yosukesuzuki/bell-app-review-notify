/**
 * Created by suzukiyosuke on 11/23/15.
 */

var assert = require('assert');

describe('patch api tests', function() {
    it('after login', function(done) {
        browser
            .url('/')
            .getText('p').then(function(value){
            assert.equal(value, 'Main Page');

            })
            .call(done);
    });
});
