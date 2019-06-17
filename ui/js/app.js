'use strict';

// the control plane URL, replaced by actual value, that is, the value of 
// of $EKSPHEMERAL_URL on container image build:
var cpURL = 'EKSPHEMERAL_URL';
// the frequency with which cluster list is refreshed given in seconds
// so  1000*60*5 for example is 5 minutes:
var refreshEvery = 1000*60*5;

$(document).ready(function($){
  // list clusters regularly:
  setInterval(clusters, refreshEvery);

  // show cluster details when user clicks 'Details':
  $('.showdetails').click(function (event) {
    event.stopPropagation();
    event.stopImmediatePropagation();
    cID = $(this).parent().attr('id');
    clusterdetail(cID);
  });
});

function clusters(){
  var ep = '/status/*';
  $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
  $.ajax({
    type: "GET",
    url: cpURL + ep,
    dataType: 'json',
    async: false,
    error: function (d) {
      console.info(d);
      $('#status').html('<div>control plane seems down</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        var consoleURL = "https://console.aws.amazon.com/eks/home";
        for (let i = 0; i < d.length; i++) {
          var cID = d[i];
          buffer += '<div id="' + cID + '">';
          buffer += ' <a href="' + consoleURL + '" target="_blank" rel="noopener">' + cID + '</a>';
          buffer += ' <span class="showdetails">Details…</span>';
          buffer += '<div class="cdetails"></div>';
          buffer += '</div>';
        }
        $('#clusterdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}

function clusterdetail(cID) {
  var ep = '/status/'+cID;
  $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
  $.ajax({
    type: "GET",
    url: cpURL + ep,
    dataType: 'json',
    async: false,
    error: function (d) {
      console.info(d);
      $('#status').html('<div>control plane seems down</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        buffer += '<div class="cdfield">TTL: ' + d.TTL + '</div>';
        buffer += '<div class="cdfield">Number of worker nodes: ' + d.numworkers + '</div>';
        $('#' + cID).$('.cdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}