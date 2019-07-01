import React, { Component } from 'react'
import ReactFauxDOM from 'react-faux-dom'
import * as d3 from 'd3'
import * as d3queue from 'd3-queue'

class Map extends Component {
	state = {
		graph: {
			nodes: [],
			links: []
		}
	}

componentWillMount() {
	d3queue.queue().defer(d3.json, "graph.json").await((error, graph) => { this.setState({graph}) })
}

renderD3() {

const el = ReactFauxDOM.createElement('svg');

var svg = d3.select(el)
var width = 640
var height = 480

svg.attr("width", width)
svg.attr("height", height)

var color = d3.scaleOrdinal(d3.schemeCategory10);

var simulation = d3.forceSimulation()
    .force("link", d3.forceLink().id(function(d) { return d.id; }))
    .force("charge", d3.forceManyBody())
    .force("center", d3.forceCenter(width / 2, height / 2));

	  var link = svg.append("g")
	    .attr("class", "links")
	    .selectAll("line")
	    .data(this.state.graph.links)
	    .enter().append("line");

	  var node = svg.append("g")
	    .attr("class", "nodes")
	    .selectAll("g")
	    .data(this.state.graph.nodes)
	    .enter().append("g");
	    // .attr("fill", function(d) { return color(d.group); })
	    
	  node.append("circle")
	      .attr("r", 5)
	      .attr("fill", function(d) { return color(d.group); })

	  node.append("text")
	      .text(function(d) {
	              return d.id;
	            })
	      .attr('x', 6)
	      .attr('y', 3);

	  node.append("title")
	      .text(function(d) { return d.id; });

	  simulation
	      .nodes(this.state.graph.nodes)

	  simulation.force("link")
	      .links(this.state.graph.links);


return el.toReact()

}

render() {
	return this.renderD3()
}

}

export default Map
