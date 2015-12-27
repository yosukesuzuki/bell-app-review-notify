/**
 * Created by suzukiyosuke on 11/23/15.
 */

var assert = require('assert');

describe('patch api tests', function() {
    it('login', function(done) {
        browser
            .url('/edit/')
            .click('#admin')
            .click('#submit-login')
            .call(done);
    });
    it('after login', function(done) {
        browser
            .url('/edit/')
            .getText('p').then(function(value){
            assert.equal(value, 'Main Page');

            })
            .call(done);
    });
});
