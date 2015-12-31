def generate_go_setting():
    tsv_file = open("appstoresettings.tsv", "r").read()
    for line in tsv_file.split("\n"):
        columns = line.split("\t")
        print('"%s":AppStoreID{CountryDomain:"%s", CountryName:"%s", CountryCode:"%s"},' % (
        columns[0].lower(), columns[0].lower(), columns[1], columns[2].strip()))


if __name__ == '__main__':
    generate_go_setting()
