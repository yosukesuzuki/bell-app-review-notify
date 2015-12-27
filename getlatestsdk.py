#!/usr/bin/env python
# -*- coding: utf-8 -*-
# https://www.googleapis.com/storage/v1/b/appengine-sdks/o?prefix=featured

import json
import re
from urllib2 import urlopen, URLError, HTTPError

SDK_URL = 'https://www.googleapis.com/storage/v1/b/appengine-sdks/o?prefix=featured'

REGEX_SDK = re.compile(r'go_appengine_sdk_linux.*\.zip')


def get_latest_sdk_url():
    url_list = []
    f = urlopen(SDK_URL)
    sdks = json.loads(f.read())
    for u in sdks['items']:
        if REGEX_SDK.search(u['id']):
            url_list.append(u['mediaLink'])
    url_list.reverse()
    if len(url_list) > 0:
        return url_list[0]
    return False


def dlfile(url):
    # Open the url
    try:
        f = urlopen(url)
        print("downloading " + url)

        # Open our local file for writing
        with open("google_appengine.zip", "wb") as local_file:
            local_file.write(f.read())
    except:
        print("Download failed")


def main():
    # Iterate over image ranges
    url = get_latest_sdk_url()
    if url:
        dlfile(url)


if __name__ == '__main__':
    main()
