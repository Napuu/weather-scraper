future_date=$(date -d "$(date) +$1 hours")
hour=$(date -d "$future_date" +%H)
day=$(date -d "$future_date" +%d)
month=$(date -d "$future_date" +%m)
year=$(date -d "$future_date" +%Y)
# forgot that I'm parsing flags as strings at Go so bc is needed remove leading zeros :-)
./scraper download --hour $(bc<<<$hour) --day $(bc<<<$day) --month $(bc<<<$month) --year $year --output temp.grb2
./scraper process --input temp.grb2 --output temp.jpeg
rm temp.grb2
mv temp.jpeg "files/${year}-${month}-${day}T${hour}:00:00.jpeg"
