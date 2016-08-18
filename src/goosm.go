package main

import (
	"encoding/xml"
	"flag"
	"github.com/cheggaaa/pb"
	"labix.org/v2/mgo"
	//"labix.org/v2/mgo/bson"
	"log"
	"os"
	"runtime"
	"time"
)

var fileName = flag.String("f", "", "OSM XML file to import.")
var mongoAddr = flag.String("s", "127.0.0.1:27017", "MongoDB server to import to.")
var mongoDbName = flag.String("db", "osm", "MongoDB database name (defaults to 'osm'.")
var mongoSession *mgo.Session
var insertChan chan interface{}
var excludeMap map[string]bool
var highwayFilterMap map[string]bool
var excludeCount int64
var onlyRoads = flag.Bool("onlyRoads", true, "Import only roads, ignore other data (eg railways, water, etc).")

//   <node id="2031042144" version="1" timestamp="2012-11-24T23:19:36Z" uid="560392" user="HostedDinner" changeset="14021355" lat="25.5805168" lon="-80.3562449"/>
type OsmNode struct {
	Id  int64 `bson:"_id" xml:"id,attr"`
	Loc struct {
		Type        string    `bson:"type"`
		Coordinates []float64 `bson:"coordinates"`
	} `bson:"loc"`
	Version   int       `bson:"ver"       xml:"version,attr"`
	Ts        time.Time `bson:"ts"        xml:"timestamp,attr"`
	Uid       int64     `bson:"uid"       xml:"uid,attr"`
	User      string    `bson:"user"      xml:"user,attr"`
	ChangeSet int64     `bson:"changeset" xml:"changeset,attr"`
	Lat       float64   `bson:"-"         xml:"lat,attr"`
	Lng       float64   `bson:"-"         xml:"lon,attr"`
}

/*
  <way id="11137619" version="2" timestamp="2013-02-05T23:54:16Z" uid="451693" user="bot-mode" changeset="14928391">
    <nd ref="99193738"/>
    <nd ref="99193742"/>
    <nd ref="99193745"/>
    <nd ref="99193748"/>
    <nd ref="99193750"/>
    <nd ref="99147506"/>
    <tag k="highway" v="residential"/>
    <tag k="name" v="Southwest 148th Avenue Court"/>
    <tag k="tiger:cfcc" v="A41"/>
    <tag k="tiger:county" v="Miami-Dade, FL"/>
    <tag k="tiger:name_base" v="148th Avenue"/>
    <tag k="tiger:name_direction_prefix" v="SW"/>
    <tag k="tiger:name_type" v="Ct"/>
    <tag k="tiger:reviewed" v="no"/>
    <tag k="tiger:zip_left" v="33185"/>
    <tag k="tiger:zip_right" v="33185"/>
  </way>*/
type OsmWay struct {
	Id  int64 `bson:"_id" xml:"id,attr"`
	Loc struct {
		Type        string      `bson:"type"`
		Coordinates [][]float64 `bson:"coordinates"`
	} `bson:"loc"`
	Version   int               `bson:"ver"       xml:"version,attr"`
	Ts        time.Time         `bson:"ts"        xml:"timestamp,attr"`
	Uid       int64             `bson:"uid"       xml:"uid,attr"`
	User      string            `bson:"user"      xml:"user,attr"`
	ChangeSet int64             `bson:"changeset" xml:"changeset,attr"`
	Tags      map[string]string `bson:"tags"`
	RTags     []struct {
		Key   string `bson:"-" xml:"k,attr"`
		Value string `bson:"-" xml:"v,attr"`
	} `bson:"-" xml:"tag"`
	Nds []struct {
		Id int64 `bson:"-" xml:"ref,attr"`
	} `bson:"-"         xml:"nd"`
	Nodes []int64 `bson:"nodes"`
}

func main() {
	var err error

	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())
	
	excludeMap = make(map[string]bool)
	highwayFilterMap = make(map[string]bool)
		
	if onlyRoads==nil || *onlyRoads {
		excludeMap["natural"]=true
		excludeMap["amenity"]=true
		excludeMap["building"]=true
		excludeMap["landuse"]=true
		excludeMap["waterway"]=true
		excludeMap["leisure"]=true
		excludeMap["railway"]=true
		excludeMap["power"]=true
		
		highwayFilterMap["service"]=true
		highwayFilterMap["footway"]=true
		highwayFilterMap["path"]=true
	}
	

	mongoSession, err = mgo.Dial(*mongoAddr)
	if err != nil {
		log.Fatalln("Can't connect to MongoDB: " + err.Error())
	}

	index := mgo.Index{
		Key: []string{"$2dsphere:loc"},
	}
	
	log.Println("Preparing database & collections...")
	
	//mongoSession.DB(*mongoDbName+"_nodes").DropDatabase()
	//mongoSession.DB(*mongoDbName+"_ways").DropDatabase()
	
//	mongoSession.DB(*mongoDbName+"_nodes").C("data").EnsureIndex(index)
	mongoSession.DB(*mongoDbName+"_ways").C("data").EnsureIndex(index)

	file, err := os.Open(*fileName)
	if err != nil {
		log.Fatalln("Can't open OSM file: " + err.Error())
	}

	insertChan = make(chan interface{}, 100)
	go goInsert()
	go goInsert()
	go goInsert()
	go goInsert()

	decoder := xml.NewDecoder(file)

	log.Println("Processing Nodes...")
	stat, _ := file.Stat()
	bar := pb.New(int(stat.Size() / 1024)).SetUnits(pb.U_NO)
	bar.Start()
	
	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}

		switch typedToken := token.(type) {
		case xml.StartElement:
			if typedToken.Name.Local == "node" {
				var n OsmNode
				decoder.DecodeElement(&n, &typedToken)
				insertChan <- n
			}
		}
		ofs, _ := file.Seek(0, 1)
		bar.Set((int)(ofs / 1024))
	}

	file.Seek(0, 0)

	decoder = xml.NewDecoder(file)

	log.Println("Processing Ways...")
	bar = pb.New(int(stat.Size() / 1024)).SetUnits(pb.U_NO)
	bar.Start()

	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}

		switch typedToken := token.(type) {
		case xml.StartElement:
			if typedToken.Name.Local == "way" {
				var w OsmWay
				err = decoder.DecodeElement(&w, &typedToken)
				if err!=nil {
					log.Printf("%s\n",err.Error())
				}
				insertChan <- w
			}
		}
		ofs, _ := file.Seek(0, 1)
		bar.Set((int)(ofs / 1024))
	}
	log.Printf("Excluded %d ways.\n",excludeCount)
	
	return
}

func goInsert() {
	sess := mongoSession.Clone()

	for {
		select {
		case i := <-insertChan:
			switch o := i.(type) {
			case OsmNode:
				o.Loc.Type = "Point"
 				o.Loc.Coordinates = []float64{o.Lng, o.Lat}
				err := sess.DB(*mongoDbName+"_nodes").C("data").Insert(o)
 				if err != nil {
 					log.Println(err.Error())
 				}
			case OsmWay:
				var n OsmNode
				exclude := false
				o.Loc.Type = "LineString"
				o.Loc.Coordinates = make([][]float64,0,len(o.Nds))
				o.Nodes = (make([]int64,0,len(o.Nds)))
				for _, nid := range o.Nds {
					if  sess.DB(*mongoDbName+"_nodes").C("data").FindId(nid.Id).One(&n)==nil {
						o.Loc.Coordinates = append(o.Loc.Coordinates,[]float64{n.Loc.Coordinates[0],n.Loc.Coordinates[1]})
					}
					o.Nodes = append(o.Nodes,nid.Id)
				}
				if len(o.Loc.Coordinates)<=1 {
					exclude = true
				}
				if len(o.RTags)==0 {
					exclude = true
				}
				o.Tags = make(map[string]string)	
				for _, t := range o.RTags {
					if _, found := excludeMap[t.Key];found {
						exclude = true
					}
					o.Tags[t.Key]=t.Value
				}
				if hwy, found := o.Tags["highway"];found {
					if _, found := highwayFilterMap[hwy];found {
						exclude = true
					}
				} else {
					exclude = true
				}
				if exclude {
					excludeCount++
					continue
				}
				
				err :=  sess.DB(*mongoDbName+"_ways").C("data").Insert(o)
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}
}
