import React from "react";
import PropTypes from "prop-types";
import {
  DetailHeadingBox,
  VideoDetailsContainer,
  DetailsTitle,
  DetailsHeading,
  DetailsTop,
  AlphaTag,
  ViewerTag,
} from "../styles/videoDetailsStyles";
import { ITGLogoURL } from "../assets/constants";

const VideoDetails = ({ viewers }) => {
  return (
    <VideoDetailsContainer>
      <DetailsTop>
        <AlphaTag>
          <i className="fas fa-construction badge-icon"></i>
          <span>Alpha</span>
        </AlphaTag>
        <ViewerTag>
          <i className="fas fa-user-friends"></i>
          <span>{viewers}</span>
        </ViewerTag>
      </DetailsTop>
      <DetailHeadingBox>
        <DetailsTitle>
          <DetailsHeading>Welcome to Project ITG</DetailsHeading>
        </DetailsTitle>
        <img id="detail-img" src={ITGLogoURL}></img>
      </DetailHeadingBox>
    </VideoDetailsContainer>
  );
};

export default VideoDetails;

VideoDetails.propTypes = {
  viewers: PropTypes.number,
};
